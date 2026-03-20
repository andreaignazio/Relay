package main

import (
	"context"
	"fmt"
	"gokafka/database"
	"gokafka/internal/gormrepo"
	"gokafka/internal/kafkaclient/consumer"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/kafkaconn"
	messagerouter "gokafka/internal/router"
	"gokafka/internal/rteventshandler"
	"gokafka/internal/shared"
	"gokafka/internal/topics"
	"gokafka/internal/workspaces"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
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

	SIGINT := syscall.SIGINT
	SIGTERM := syscall.SIGTERM

	ctx, cancel := signal.NotifyContext(context.Background(), SIGINT, SIGTERM)
	defer cancel()
	brokers := []string{"localhost:9092"}

	conn, err := kafkaconn.NewKafkaConn("localhost:9092", brokers)
	if err != nil {
		panic(err)
	}

	commandTopic := shared.Topic{
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

	topics.CreateTopic(commandTopic, conn.Conn)
	topics.CreateTopic(eventsTopic, conn.Conn)

	commandRouter := messagerouter.NewCommandRouter()
	eventRouter := messagerouter.NewEventRouter()

	eventProducer := producer.NewKafkaProducer(eventsTopic, conn)

	gormRepository := gormrepo.NewRepository(db)
	workspaceService := workspaces.NewService(gormRepository, eventProducer, logChan)
	rtEventsHandler := rteventshandler.NewHandler(logChan)

	commandRouter.RegisterHandler(shared.ActionKeyWorkspaceCreate, workspaceService.HandleCreateWorkspace)
	eventRouter.RegisterHandler(shared.ActionKeyWorkspaceCreate, rtEventsHandler.HandleCreateWorkspaceEvent)

	commandConsumer := consumer.NewKafkaConsumer(commandTopic, shared.CommandConsumerGroupId, conn)
	viewMaterializerConsumer := consumer.NewKafkaConsumer(eventsTopic, shared.ViewMaterializerConsumerGroupId, conn)
	realTimeHandlerConsumer := consumer.NewKafkaConsumer(eventsTopic, shared.RealTimeHandlerConsumerGroupId, conn)

	consumersByGroupId := map[shared.ConsumerGroupId][]*consumer.KafkaConsumer{
		commandConsumer.GroupID:          {commandConsumer},
		viewMaterializerConsumer.GroupID: {viewMaterializerConsumer},
		realTimeHandlerConsumer.GroupID:  {realTimeHandlerConsumer},
	}

	routerByConsumerGroupId := map[shared.ConsumerGroupId]messagerouter.Router{
		commandConsumer.GroupID:          *commandRouter,
		viewMaterializerConsumer.GroupID: *eventRouter,
		realTimeHandlerConsumer.GroupID:  *eventRouter,
	}

	commandProducer := producer.NewKafkaProducer(commandTopic, conn)

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cmd := shared.NewCommand(
					shared.ActionKeyWorkspaceCreate,
					shared.NewWorkspaceCommandPartitionKey(uuid.New().String()),
					map[string]string{"name": "Test Workspace"},
				)
				if err := commandProducer.WriteMessage(ctx, cmd); err != nil {
					fmt.Println("mock producer error:", err)
				}
			}
		}
	}()

	var errChan = make(chan error, 100)

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
