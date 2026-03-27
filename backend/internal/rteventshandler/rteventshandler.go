package rteventshandler

import (
	"context"
	"fmt"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/shared"
	"gokafka/internal/websocketcommands"
)

type Handler struct {
	WorkspaceTopicProducer *producer.KafkaProducer
	ChannelTopicProducer   *producer.KafkaProducer
	UserTopicProducer      *producer.KafkaProducer
	SnapshotRepo           SnapshotRepo
	logChan                chan string
}

func NewHandler(
	logChan chan string,
	workspaceTopicProducer *producer.KafkaProducer,
	channelTopicProducer *producer.KafkaProducer,
	userTopicProducer *producer.KafkaProducer,
	snapshotRepo SnapshotRepo,
) *Handler {
	return &Handler{
		logChan:                logChan,
		WorkspaceTopicProducer: workspaceTopicProducer,
		ChannelTopicProducer:   channelTopicProducer,
		UserTopicProducer:      userTopicProducer,
		SnapshotRepo:           snapshotRepo,
	}
}

func (h *Handler) HandleEvent(ctx context.Context, event shared.Event) error {
	h.logChan <- fmt.Sprintf("[RTHandler] ActionKey: %s", event.GetActionKey())

	routes, ok := routeTable[event.GetActionKey()]
	if !ok {
		h.logChan <- fmt.Sprintf("[RTHandler] no routes for ActionKey: %s", event.GetActionKey())
		return nil
	}

	for _, route := range routes {
		entityKey, ok := topicToEntity[route.Topic]
		if !ok {
			return fmt.Errorf("no entity key for topic: %s", route.Topic)
		}
		partitionIDs, ok := event.EntityContext[entityKey]
		if !ok {
			return fmt.Errorf("EntityContext missing key %s for ActionKey %s", entityKey, event.GetActionKey())
		}

		payload, err := h.buildPayload(ctx, event, route.EventType)
		if err != nil {
			return fmt.Errorf("build payload (%s/%s): %w", route.Topic, route.EventType, err)
		}
		if payload == nil {
			continue
		}

		for _, partitionID := range partitionIDs {
			rtEvent := shared.RTEvent{
				MessageID:    event.GetMessageID(),
				AggregateID:  event.GetAggregateID(),
				ActionKey:    event.GetActionKey(),
				Type:         route.EventType,
				PartitionKey: partitionKeyForTopic(route.Topic, partitionID),
				Payload:      payload,
			}
			if err := h.producerForTopic(route.Topic).WriteMessage(ctx, rtEvent); err != nil {
				return fmt.Errorf("write rt event (%s): %w", route.Topic, err)
			}
		}
	}

	commandAck := websocketcommands.CommandAck{
		ActionKey: event.GetActionKey(),
		MessageID: event.GetMessageID(),
		Success:   true,
	}
	commandAckBytes, err := commandAck.Bytes()
	if err != nil {
		return fmt.Errorf("marshal command ack: %w", err)
	}
	ackEvent := shared.RTEvent{
		MessageID:    event.GetMessageID(),
		AggregateID:  event.GetAggregateID(),
		ActionKey:    event.GetActionKey(),
		Type:         shared.RTEventTypeCommandAck,
		PartitionKey: shared.NewUserRealTimePartitionKey(event.GetAuthorID().String()),
		Payload:      commandAckBytes,
	}

	if err := h.producerForTopic(shared.RealTimeDestinationUsers).WriteMessage(ctx, ackEvent); err != nil {
		return fmt.Errorf("write command ack event: %w", err)
	}
	h.logChan <- fmt.Sprintf("[RTHandler] Successfully processed event and sent command ack for ActionKey: %s to user: %s", event.GetActionKey(), event.GetAuthorID())

	return nil
}
