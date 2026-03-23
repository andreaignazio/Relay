package messagerouter

import (
	"context"
	"fmt"
	"gokafka/internal/shared"

	"github.com/segmentio/kafka-go"
)

type CommandRouter struct {
	handlers map[shared.ActionKey]func(ctx context.Context, command shared.Command) error
}

func NewCommandRouter() *CommandRouter {
	return &CommandRouter{
		handlers: make(map[shared.ActionKey]func(ctx context.Context, command shared.Command) error),
	}
}

func (cr *CommandRouter) RegisterHandler(actionKey shared.ActionKey, handler func(ctx context.Context, command shared.Command) error) {
	cr.handlers[actionKey] = handler
}

func (cr *CommandRouter) RouteCommand(ctx context.Context, command shared.Command) error {
	handler, exists := cr.handlers[command.ActionKey]
	if !exists {
		return fmt.Errorf("no handler registered for action key: %s", command.ActionKey)
	}
	return handler(ctx, command)
}

func (cr CommandRouter) HandleMessage(ctx context.Context, message kafka.Message) error {
	value := message.Value
	//var command shared.Command
	command, err := shared.CommandFromBytes(value)
	if err != nil {
		return err
	}
	return cr.RouteCommand(ctx, command)
}
