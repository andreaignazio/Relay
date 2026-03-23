package producer

import (
	"context"
	"gokafka/internal/kafkaconn"
	"gokafka/internal/shared"

	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	Writer *kafka.Writer
	Topic  shared.Topic
	Conn   *kafkaconn.KafkaConn
}

func NewKafkaProducer(topic shared.Topic, conn *kafkaconn.KafkaConn) *KafkaProducer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(conn.Brokers...),
		Topic:    topic.Name,
		Balancer: &kafka.Murmur2Balancer{},
	}

	return &KafkaProducer{
		Writer: writer,
		Topic:  topic,
		Conn:   conn,
	}
}

func (kp *KafkaProducer) WriteMessage(ctx context.Context, event shared.KafkaWritable) error {

	messages := make([]kafka.Message, 0, 1)
	payload, err := event.Bytes()
	if err != nil {
		return err
	}

	messages = append(messages, kafka.Message{
		Key:   event.GetPartitionKey().Bytes(),
		Value: payload,
	})

	if err := kp.Writer.WriteMessages(ctx, messages...); err != nil {
		return err
	}
	return nil
}

func (kp *KafkaProducer) Close() error {
	return kp.Writer.Close()
}
