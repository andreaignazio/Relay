package messagerouter

import (
	"context"

	"github.com/segmentio/kafka-go"
)

type Router interface {
	HandleMessage(ctx context.Context, message kafka.Message) error
	//RouteMessage(ctx context.Context, message shared.KafkaDomainMessage) error
}
