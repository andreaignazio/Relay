package topics

import (
	"gokafka/internal/shared"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

func CreateTopic(topic shared.Topic, conn *kafka.Conn) error {
	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic.Name,
		NumPartitions:     topic.Partitions,
		ReplicationFactor: topic.Replicas,
	})
	if err == kafka.TopicAlreadyExists {
		return nil
	}
	return err
}

func DeleteTopic(topic shared.Topic, conn *kafka.Conn) error {
	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	return controllerConn.DeleteTopics(topic.Name)
}
