package main

import (
	"context"
	"fmt"
	"gokafka/database"
	"gokafka/internal/api"
	"gokafka/internal/gormrepo"
	"gokafka/internal/kafkaclient/consumer"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/kafkaconn"
	messagerouter "gokafka/internal/router"
	"gokafka/internal/rteventshandler"
	"gokafka/internal/shared"
	"gokafka/internal/topics"
	"gokafka/internal/viewmaterializer"
	"gokafka/internal/websocketcommands"
	"gokafka/internal/websocketdomain"
	"gokafka/internal/workspaces"
	"gokafka/internal/wsaggregator"
	"os/signal"
	"syscall"
)

func main() {
	db := database.InitDB()
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	if err := sqlDB.Ping(); err != nil {
		panic(err)
	}
	fmt.Println("connected to database")

	if err := database.RunMigrations(db); err != nil {
		panic(err)
	}
	fmt.Println("migrations applied")

	SIGINT := syscall.SIGINT
	SIGTERM := syscall.SIGTERM

	ctx, cancel := signal.NotifyContext(context.Background(), SIGINT, SIGTERM)
	defer cancel()
	brokers := []string{"localhost:9092"}

	conn, err := kafkaconn.NewKafkaConn("localhost:9092", brokers)
	if err != nil {
		panic(err)
	}

	domainCommandTopic := shared.Topic{
		Name:       "messages.commands",
		Partitions: 3,
		Replicas:   1,
	}
	eventsTopic := shared.Topic{
		Name:       "messages.events",
		Partitions: 3,
		Replicas:   1,
	}

	logChan := make(chan string, 100)

	rtWorkspaceTopic := shared.RealTimeTopicWorkspaces.Topic(1, 1)
	rtChannelTopic := shared.RealTimeTopicChannels.Topic(1, 1)
	rtUserTopic := shared.RealTimeTopicUsers.Topic(1, 1)

	wsCommandTopic := shared.Topic{
		Name:       "messages.websocketcommands",
		Partitions: 3,
		Replicas:   1,
	}

	for _, t := range []shared.Topic{domainCommandTopic, eventsTopic, rtWorkspaceTopic, rtChannelTopic, rtUserTopic, wsCommandTopic} {
		if err := topics.CreateTopic(t, conn.Conn); err != nil {
			panic(fmt.Sprintf("failed to create topic %q: %v", t.Name, err))
		}
	}

	workspaceTopicProducer := producer.NewKafkaProducer(rtWorkspaceTopic, conn)
	channelTopicProducer := producer.NewKafkaProducer(rtChannelTopic, conn)
	userTopicProducer := producer.NewKafkaProducer(rtUserTopic, conn)
	wsCommandProducer := producer.NewKafkaProducer(wsCommandTopic, conn)

	workspaceTopicConsumer := consumer.NewKafkaConsumer(rtWorkspaceTopic, shared.RealTimeAggregatorConsumerGroupId, conn)
	channelTopicConsumer := consumer.NewKafkaConsumer(rtChannelTopic, shared.RealTimeAggregatorConsumerGroupId, conn)
	userTopicConsumer := consumer.NewKafkaConsumer(rtUserTopic, shared.RealTimeAggregatorConsumerGroupId, conn)
	wsCommandConsumer := consumer.NewKafkaConsumer(wsCommandTopic, shared.WebSocketCommandsConsumerGroupId, conn)

	ackChan := make(chan websocketcommands.AggregatorAck, 100)

	wsAggregator := wsaggregator.NewWsAggregator(workspaceTopicConsumer,
		channelTopicConsumer,
		userTopicConsumer,
		wsCommandConsumer, conn,
		logChan, make(chan error, 100), ackChan)

	commandRouter := messagerouter.NewCommandRouter()
	eventRouter := messagerouter.NewEventRouter()
	materializerRouter := messagerouter.NewEventRouter()

	eventProducer := producer.NewKafkaProducer(eventsTopic, conn)

	gormRepository := gormrepo.NewRepository(db)
	workspaceService := workspaces.NewService(gormRepository, gormRepository, gormRepository, eventProducer, logChan)
	rtEventsHandler := rteventshandler.NewHandler(logChan, workspaceTopicProducer, channelTopicProducer, userTopicProducer)
	viewmaterializerService := viewmaterializer.NewService(gormRepository, gormRepository)

	commandRouter.RegisterHandler(shared.ActionKeyWorkspaceCreate, workspaceService.HandleCreateWorkspace)
	eventRouter.RegisterHandler(shared.ActionKeyWorkspaceCreate, rtEventsHandler.HandleCreateWorkspaceEvent)

	//Materializer handlers would be registered here, for example:
	materializerRouter.RegisterHandler(shared.ActionKeyWorkspaceCreate, viewmaterializerService.HandleWorkspaceViewUpdate)

	domainCommandConsumer := consumer.NewKafkaConsumer(domainCommandTopic, shared.CommandConsumerGroupId, conn)
	viewMaterializerConsumer := consumer.NewKafkaConsumer(eventsTopic, shared.ViewMaterializerConsumerGroupId, conn)
	realTimeHandlerConsumer := consumer.NewKafkaConsumer(eventsTopic, shared.RealTimeHandlerConsumerGroupId, conn)

	consumersByGroupId := map[shared.ConsumerGroupId][]*consumer.KafkaConsumer{
		domainCommandConsumer.GroupID:    {domainCommandConsumer},
		viewMaterializerConsumer.GroupID: {viewMaterializerConsumer},
		realTimeHandlerConsumer.GroupID:  {realTimeHandlerConsumer},
	}

	routerByConsumerGroupId := map[shared.ConsumerGroupId]messagerouter.Router{
		domainCommandConsumer.GroupID:    *commandRouter,
		viewMaterializerConsumer.GroupID: *materializerRouter,
		realTimeHandlerConsumer.GroupID:  *eventRouter,
	}

	domainCommandProducer := producer.NewKafkaProducer(domainCommandTopic, conn)

	var errChan = make(chan error, 100)

	wsHub := websocketdomain.NewHub(logChan, wsCommandProducer, errChan, wsAggregator.ConnectClientChan, ackChan)

	apiHandler := api.NewHandler(domainCommandProducer)
	readerApiHandler := api.NewReaderHandler(gormRepository)
	wsHandler := api.NewWsHandler(wsHub, conn, shared.WebSocketClientsConsumerGroupId, logChan, ctx)
	server := api.NewRouter(apiHandler, readerApiHandler, wsHandler)

	for groupId, consumers := range consumersByGroupId {
		router := routerByConsumerGroupId[groupId]
		for _, c := range consumers {

			go func(c *consumer.KafkaConsumer, router messagerouter.Router) {
				if err := c.Run(ctx, router.HandleMessage); err != nil {
					errChan <- err
				}
			}(c, router)
		}
	}

	go func() {
		defer wsAggregator.Close()
		wsAggregator.Run(ctx)
	}()

	go func() {
		defer wsHub.Close()
		wsHub.Run(ctx)
	}()

	go func() {
		if err := server.Run("localhost:8080"); err != nil {
			errChan <- err
		}
	}()

	go func() {
		for log := range logChan {
			fmt.Println("log:", log)
		}

	}()

	select {
	case <-ctx.Done():
		fmt.Println("shutdown signal received")
	case err := <-errChan:
		fmt.Println("consumer error:", err)
	}

}
