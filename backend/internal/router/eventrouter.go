package messagerouter

import (
	"context"
	"gokafka/internal/shared"

	"github.com/segmentio/kafka-go"
)

type EventRouter struct {
	handlers map[shared.ActionKey]func(ctx context.Context, event shared.Event) error
}

func NewEventRouter() *EventRouter {
	return &EventRouter{
		handlers: make(map[shared.ActionKey]func(ctx context.Context, event shared.Event) error),
	}
}

func (er *EventRouter) RegisterHandler(actionKey shared.ActionKey, handler func(ctx context.Context, event shared.Event) error) {
	er.handlers[actionKey] = handler
}

func (er *EventRouter) RouteEvent(ctx context.Context, event shared.Event) error {

	handler, exists := er.handlers[event.ActionKey]
	if !exists {
		return nil // No handler registered for this action key, ignore the event
	}
	return handler(ctx, event)
}

func (er EventRouter) HandleMessage(ctx context.Context, message kafka.Message) error {
	value := message.Value
	event, err := shared.EventFromBytes(value)
	if err != nil {
		return err
	}
	return er.RouteEvent(ctx, event)
}
