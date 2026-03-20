package consumer

import (
	"context"
	"fmt"
	"gokafka/internal/kafkaconn"
	"gokafka/internal/shared"
	"time"

	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	Reader  *kafka.Reader
	Topic   shared.Topic
	GroupID shared.ConsumerGroupId
	Conn    *kafkaconn.KafkaConn
}

func NewKafkaConsumer(topic shared.Topic, groupID shared.ConsumerGroupId, conn *kafkaconn.KafkaConn) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  conn.Brokers,
		Topic:    topic.Name,
		GroupID:  groupID.String(),
		MaxBytes: 10e6,
		MinBytes: 1,
		MaxWait:  500 * time.Millisecond, // 500ms
	})

	return &KafkaConsumer{
		Reader:  reader,
		Topic:   topic,
		GroupID: groupID,
		Conn:    conn,
	}
}

func (kc *KafkaConsumer) Close() error {

	if err := kc.Reader.Close(); err != nil {

		return err
	}
	fmt.Println("consumer has closed the reader")

	return nil
}

func (kc *KafkaConsumer) Run(ctx context.Context, handle func(ctx context.Context, msg kafka.Message) error) error {

	for {
		msg, err := kc.Reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // uscita pulita per cancellazione del context
			}
			return err
		}
		if err := handle(ctx, msg); err != nil {
			return err
		}
		if err = kc.Reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
		//kc.Messages = append(kc.Messages, msg)
		//fmt.Println("message at topic/partition/offset ", msg.Topic, "/", msg.Partition, "/", msg.Offset, ": ", string(msg.Key), ":", string(msg.Value))

	}
}
