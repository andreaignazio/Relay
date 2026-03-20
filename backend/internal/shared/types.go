package shared

import "github.com/segmentio/kafka-go"

type Topic struct {
	Name       string
	Partitions int
	Replicas   int
}

type Message struct {
	Key   []byte
	Value []byte
}

type CommittedMessage struct {
	Message       kafka.Message
	ConsumerGroup string
}

type CommittedEvent struct {
	Event         Event
	Message       kafka.Message
	ConsumerGroup string
}

type ConsumerGroupId string

const (
	CommandConsumerGroupId          ConsumerGroupId = "commands"
	ViewMaterializerConsumerGroupId ConsumerGroupId = "view-materializer"
	RealTimeHandlerConsumerGroupId  ConsumerGroupId = "real-time-handler"
)

func (cg ConsumerGroupId) String() string {
	return string(cg)
}
