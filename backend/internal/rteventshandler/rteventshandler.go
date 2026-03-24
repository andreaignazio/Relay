package rteventshandler

import (
	"context"
	"fmt"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/shared"
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
		partitionID, ok := event.EntityContext[entityKey]
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

	return nil
}
