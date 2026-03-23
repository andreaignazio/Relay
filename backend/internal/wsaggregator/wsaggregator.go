// Package wsaggregator implements the Subscription Aggregator for the WebSocket Gateway.
//
// Architecture: actor pattern — a single Actor goroutine owns all maps (SubscriptionsByClient,
// ProducerByClient) and is the only goroutine that reads or writes them. Kafka consumer
// goroutines forward messages to rtChan or cmdChan; the Actor processes them sequentially.
//
// Delivery guarantee: at-most-once for RT events (MVP decision).
// Consumer goroutines commit the Kafka offset as soon as the message is forwarded to the
// channel, before the Actor processes it. If the Actor fails, the message is lost.
// This is acceptable for real-time notifications (presence, invalidations, rich events)
// because the frontend reconciles state on the next interaction. A future improvement
// would introduce an ack channel to achieve at-least-once delivery.
package wsaggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"gokafka/internal/kafkaclient/consumer"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/kafkaconn"
	"gokafka/internal/shared"
	"gokafka/internal/websocketcommands"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

type WsAggregator struct {
	WorkspaceConsumer *consumer.KafkaConsumer
	ChannelConsumer   *consumer.KafkaConsumer
	UserConsumer      *consumer.KafkaConsumer
	WsCommandConsumer *consumer.KafkaConsumer
	KafkaConn         *kafkaconn.KafkaConn
	GlobalErrChan     chan error
	LogChan           chan string

	SubscriptionsByClient map[uuid.UUID][]websocketcommands.Subscription // Map of client ID to list of subscribed topics
	ProducerByClient      map[uuid.UUID]*producer.KafkaProducer          // Map of client ID to Kafka producer for sending messages
	rtChan                chan kafka.Message
	cmdChan               chan kafka.Message
	ConnectClientChan     chan websocketcommands.WsCommandMessage
	ackChan               chan websocketcommands.AggregatorAck
	errChan               chan error

	// Add fields as needed for your aggregator, such as channels, state, etc.
}

func NewWsAggregator(WorkspaceConsumer, ChannelConsumer, UserConsumer, WsCommandConsumer *consumer.KafkaConsumer, KafkaConn *kafkaconn.KafkaConn, logChan chan string, ErrChan chan error, ackChan chan websocketcommands.AggregatorAck) *WsAggregator {
	return &WsAggregator{
		WorkspaceConsumer:     WorkspaceConsumer,
		ChannelConsumer:       ChannelConsumer,
		UserConsumer:          UserConsumer,
		WsCommandConsumer:     WsCommandConsumer,
		KafkaConn:             KafkaConn,
		GlobalErrChan:         ErrChan,
		LogChan:               logChan,
		SubscriptionsByClient: make(map[uuid.UUID][]websocketcommands.Subscription),
		ConnectClientChan:     make(chan websocketcommands.WsCommandMessage, 100),
		ProducerByClient:      make(map[uuid.UUID]*producer.KafkaProducer),
		rtChan:                make(chan kafka.Message, 100),
		cmdChan:               make(chan kafka.Message, 100),
		ackChan:               ackChan,
		errChan:               make(chan error, 100),
	}
}

func (wa *WsAggregator) Run(ctx context.Context) {
	go wa.Actor(ctx)
	go func() {
		if err := wa.WorkspaceConsumer.Run(ctx, wa.HandleRtMessage); err != nil {
			wa.GlobalErrChan <- err
		}
	}()
	go func() {
		if err := wa.ChannelConsumer.Run(ctx, wa.HandleRtMessage); err != nil {
			wa.GlobalErrChan <- err
		}
	}()
	go func() {
		if err := wa.UserConsumer.Run(ctx, wa.HandleRtMessage); err != nil {
			wa.GlobalErrChan <- err
		}
	}()
	go func() {
		if err := wa.WsCommandConsumer.Run(ctx, wa.HandleCommandMessage); err != nil {
			wa.GlobalErrChan <- err
		}
	}()

}

func (wa *WsAggregator) Actor(ctx context.Context) {
	for {
		select {
		case msg := <-wa.rtChan:
			wa.LogChan <- "[WSAggregator]Received RT message for aggregation: " + string(msg.Value) + " with key: " + string(msg.Key)
			if err := wa.handleMessage(ctx, msg); err != nil {
				wa.GlobalErrChan <- err
			}
		case cmdMsg := <-wa.cmdChan:
			if err := wa.HandleWsCommand(ctx, cmdMsg); err != nil {

				wa.GlobalErrChan <- err
			}
		case connectClientCmd := <-wa.ConnectClientChan:
			if err := wa.HandleConnectCommand(ctx, connectClientCmd); err != nil {
				wa.GlobalErrChan <- err
			}
			wa.LogChan <- fmt.Sprintf("[WSAggregator] Registered new client with command: %s", string(connectClientCmd.Payload))
		case <-ctx.Done():
			return
		}
	}

}

func (wa *WsAggregator) HandleRtMessage(ctx context.Context, msg kafka.Message) error {
	select {
	case wa.rtChan <- msg:
	case <-ctx.Done():
	}
	return nil
}

func (wa *WsAggregator) HandleCommandMessage(ctx context.Context, msg kafka.Message) error {
	select {
	case wa.cmdChan <- msg:
	case <-ctx.Done():
	}
	return nil
}

func (wa *WsAggregator) handleMessage(ctx context.Context, msg kafka.Message) error {
	partitionKeyStr := string(msg.Key)
	if partitionKeyStr == "" {
		return nil
	}
	parsed, err := shared.ParsePartitionKey(partitionKeyStr)
	if err != nil {
		wa.LogChan <- "Error parsing partition key: " + err.Error()
		return nil
	}
	partitionKey, ok := parsed.(shared.RealTimePartitionKey)
	if !ok {
		// Handle error or log it
		return nil
	}
	entityID, err := uuid.Parse(partitionKey.EntityID)
	if err != nil {
		// Handle error or log it
		return nil
	}
	messageRTDestination := partitionKey.GetRealTimeDestination()

	for clientID, subscriptions := range wa.SubscriptionsByClient {
		for _, subscription := range subscriptions {
			subscriptiorRTDestination, err := subscription.RealTimeTopic.GetRealTimeDestination()
			if err != nil {
				wa.LogChan <- "Error parsing real-time topic: " + err.Error()
				continue
			}
			if subscriptiorRTDestination == messageRTDestination {
				if subscription.ID == entityID {
					// Send message to client
					message, err := wa.AggregateMessage(ctx, clientID, msg)
					if err != nil {
						// Handle error or log it
						wa.LogChan <- "Error aggregating message: " + err.Error()
						continue
					}
					if err := wa.GetProducer(ctx, clientID).WriteMessage(ctx, message); err != nil {
						// Handle error or log it
						wa.LogChan <- "Error writing message: " + err.Error()
						continue
					}
					wa.LogChan <- fmt.Sprintf("[WSAggregator] Sent message to client %s for subscription %s with entity ID %s", clientID, subscription.RealTimeTopic, subscription.ID)
				}
			}
		}
	}

	return nil
}

func (wa *WsAggregator) AggregateMessage(ctx context.Context, clientID uuid.UUID, msg kafka.Message) (*shared.WsAggregatedEvent, error) {
	rtevent, err := shared.RTEventFromBytes(msg.Value)
	if err != nil {
		return nil, err
	}
	clientIDStr := clientID.String()
	partitionKey := shared.NewWsAggregatedPartitionKey(clientIDStr)
	aggregatedEvent := &shared.WsAggregatedEvent{
		RTEvent:      rtevent,
		PartitionKey: partitionKey,
	}
	return aggregatedEvent, nil
}

func (wa *WsAggregator) GetProducer(ctx context.Context, clientID uuid.UUID) *producer.KafkaProducer {
	if producer, exists := wa.ProducerByClient[clientID]; exists {
		return producer
	}
	producer := wa.MakeProducer(ctx, clientID)
	wa.ProducerByClient[clientID] = producer
	return producer
}

func (wa *WsAggregator) MakeProducer(ctx context.Context, clientID uuid.UUID) *producer.KafkaProducer {
	topic := shared.NewWsClientTopic(clientID.String(), 1, 1)
	return producer.NewKafkaProducer(topic, wa.KafkaConn)
}

func (wa *WsAggregator) HandleWsCommand(ctx context.Context, msg kafka.Message) error {
	fmt.Println("[WSAggregator] Handling WS command message: " + string(msg.Value) + " with key: " + string(msg.Key))
	command, err := websocketcommands.WsCommandMessageFromBytes(msg.Value)
	if err != nil {
		fmt.Println("[WSAggregator] Error parsing WS command message: " + err.Error())
		return err
	}
	switch command.WsCommandKey {
	case websocketcommands.WsCommandKeySubscribe:
		var payload websocketcommands.SubscribeCommandPayload
		if err := json.Unmarshal(command.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal subscribe command payload: %w", err)
		}
		clientID := payload.ClientID
		if _, exists := wa.SubscriptionsByClient[clientID]; !exists {
			wa.SubscriptionsByClient[clientID] = []websocketcommands.Subscription{}
		}
		wa.SubscriptionsByClient[clientID] = append(wa.SubscriptionsByClient[clientID], payload.Subscriptions...)
		topic := shared.NewWsClientTopic(clientID.String(), 1, 1)
		wa.ProducerByClient[clientID] = producer.NewKafkaProducer(topic, wa.KafkaConn)
		wa.sendAck(clientID, websocketcommands.WsCommandKeySubscribe, true, "")
		wa.LogChan <- fmt.Sprintf("[WSAggregator] Subscribed client %s to topics: %v", clientID, payload.Subscriptions)

	case websocketcommands.WsCommandKeyUnsubscribe:
		var payload websocketcommands.UnsubscribeCommandPayload
		if err := json.Unmarshal(command.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal unsubscribe command payload: %w", err)
		}
		clientID := payload.ClientID
		if _, exists := wa.SubscriptionsByClient[clientID]; exists {
			// Remove the specified subscriptions for the client
			subscriptions := wa.SubscriptionsByClient[clientID]
			unSubSet := make(map[websocketcommands.Subscription]struct{})
			for _, unsub := range payload.Subscriptions {
				unSubSet[unsub] = struct{}{}
			}
			newSubscriptions := []websocketcommands.Subscription{}
			for _, sub := range subscriptions {
				if _, shouldUnsub := unSubSet[sub]; !shouldUnsub {
					newSubscriptions = append(newSubscriptions, sub)
				}
			}
			wa.SubscriptionsByClient[clientID] = newSubscriptions
			if len(newSubscriptions) == 0 {
				wa.CleanUpClientResources(clientID)
			}
		}
		wa.sendAck(clientID, websocketcommands.WsCommandKeyUnsubscribe, true, "")

	case websocketcommands.WsCommandKeyDisconnect:
		var payload websocketcommands.DisconnectCommandPayload
		if err := json.Unmarshal(command.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal disconnect command payload: %w", err)
		}
		clientID := payload.ClientID

		wa.CleanUpClientResources(clientID)

		// Handle disconnect command
	case websocketcommands.WsCommandKeyFocus:
		var payload websocketcommands.FocusCommandPayload
		if err := json.Unmarshal(command.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal focus command payload: %w", err)
		}
		wa.sendAck(payload.ClientID, websocketcommands.WsCommandKeyFocus, true, "")
	case websocketcommands.WsCommandKeyConnect:
		wa.HandleConnectCommand(ctx, *command)
	default:
		wa.LogChan <- "Unknown command key: " + string(command.WsCommandKey)
	}
	return nil
}

func (wa *WsAggregator) CleanUpClientResources(clientID uuid.UUID) {
	delete(wa.SubscriptionsByClient, clientID)
	//delete(wa.AggregateTopicByClient, clientID)
	if producer, exists := wa.ProducerByClient[clientID]; exists {
		producer.Close()
		delete(wa.ProducerByClient, clientID)
	}
}

func (wa *WsAggregator) sendAck(clientID uuid.UUID, key websocketcommands.WsCommandKey, success bool, errMsg string) {
	select {
	case wa.ackChan <- websocketcommands.AggregatorAck{
		ClientID: clientID,
		Ack: websocketcommands.WsCommandAck{
			WsCommandKey: key,
			Success:      success,
			Error:        errMsg,
		},
	}:
	default:
		wa.LogChan <- "[WSAggregator] ackChan full, dropping ack for " + clientID.String()
	}
}

func (wa *WsAggregator) Close() {
	for _, producer := range wa.ProducerByClient {
		producer.Close()
	}
}

func (wa *WsAggregator) HandleConnectCommand(ctx context.Context, command websocketcommands.WsCommandMessage) error {
	fmt.Println("[WSAggregator] Handling Connect command")
	var payload websocketcommands.ConnectCommandPayload
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal connect command payload: %w", err)
	}
	clientID := payload.ClientID
	userID := payload.UserID
	wa.SubscriptionsByClient[clientID] = []websocketcommands.Subscription{}
	//producer := wa.GetProducer(ctx, clientID)
	topic := shared.RealTimeTopicUsers
	//topics.CreateTopic(topic, wa.KafkaConn.Conn)

	userTopicSubscription := websocketcommands.Subscription{
		RealTimeTopic: topic,
		ID:            userID,
	}
	wa.LogChan <- fmt.Sprintf("[WSAggregator] Subscribed client %s to user topic %s with entity ID %s", clientID, topic, userID)
	wa.SubscriptionsByClient[clientID] = append(wa.SubscriptionsByClient[clientID], userTopicSubscription)
	wa.ProducerByClient[clientID] = producer.NewKafkaProducer(shared.NewWsClientTopic(clientID.String(), 1, 1), wa.KafkaConn)
	return nil
}
