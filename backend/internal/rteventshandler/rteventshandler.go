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
	logChan                chan string
	// Define any dependencies or fields needed for the handler
}

func NewHandler(logChan chan string, workspaceTopicProducer *producer.KafkaProducer, channelTopicProducer *producer.KafkaProducer, userTopicProducer *producer.KafkaProducer) *Handler {
	return &Handler{
		logChan:                logChan,
		WorkspaceTopicProducer: workspaceTopicProducer,
		ChannelTopicProducer:   channelTopicProducer,
		UserTopicProducer:      userTopicProducer,
	}
}

func (h *Handler) HandleEvent(ctx context.Context, event shared.Event) error {
	routes := h.GetEventRoutes(event)
	for _, route := range routes {
		rtEvent, err := route.EventBuilder(ctx, event)
		if err != nil {
			return fmt.Errorf("build rt event: %w", err)
		}
		if rtEvent == nil {
			continue
		}
		rtEvent.PartitionKey = route.PartionFn(event)
		if err := route.Procucer.WriteMessage(ctx, *rtEvent); err != nil {
			return fmt.Errorf("write rt event: %w", err)
		}
	}
	return nil
}

func (h *Handler) HandleRichEvent(ctx context.Context, event shared.Event) (*shared.RTEvent, error) {
	//fmt.Println("[RTHandler]Handling rich event with ActionKey:", event.GetActionKey())
	h.logChan <- fmt.Sprintf("[RTHandler]Handling rich event with ActionKey: %s", event.GetActionKey())
	return nil, nil
}

func (h *Handler) HandleInvalidateCacheEvent(ctx context.Context, event shared.Event) (*shared.RTEvent, error) {
	//fmt.Println("[RTHandler]Handling InvalidateCache event for aggregate ID:", event.GetAggregateID())
	h.logChan <- fmt.Sprintf("[RTHandler]Handling InvalidateCache event for aggregate ID: %s", event.GetAggregateID())
	rtEvent := &shared.RTEvent{
		MessageID:   event.GetMessageID(),
		AggregateID: event.GetAggregateID(),
		ActionKey:   event.GetActionKey(),
		Type:        shared.RTEventTypeInvalidateCache,
		Payload:     event.Payload,
	}
	return rtEvent, nil
}

func (h *Handler) HandleHintEvent(ctx context.Context, event shared.Event) (*shared.RTEvent, error) {
	//fmt.Println("[RTHandler]Handling Hint event with ActionKey:", event.GetActionKey())
	h.logChan <- fmt.Sprintf("[RTHandler]Handling Hint event with ActionKey: %s", event.GetActionKey())
	return nil, nil
}

func (h *Handler) HandleCreateWorkspaceEvent(ctx context.Context, event shared.Event) error {
	h.logChan <- fmt.Sprintf("[RTHandler]Handling CreateWorkspace event for workspace ID: %s", event)
	return h.HandleEvent(ctx, event)
	//fmt.Println("[RTHandler]Handling CreateWorkspace event for workspace ID:", event.GetPartitionKey())

	return nil
}
