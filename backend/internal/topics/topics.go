package topics

import (
	"gokafka/internal/shared"
	"net"
	"strconv"

	"github.com/segmentio/kafka-go"
)

func CreateTopic(topic shared.Topic, conn *kafka.Conn) {

	controller, err := conn.Controller()
	if err != nil {
		panic(err.Error())
	}

	var controllerConn *kafka.Conn
	controllerConn, err = kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		panic(err.Error())
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic.Name,
			NumPartitions:     topic.Partitions,
			ReplicationFactor: topic.Replicas,
		},
	}

	err = controllerConn.CreateTopics(topicConfigs...)
	if err != nil {
		panic(err.Error())
	}

}

func DeleteTopic(topic shared.Topic, conn *kafka.Conn) {
	controller, err := conn.Controller()
	if err != nil {
		panic(err.Error())
	}

	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		panic(err.Error())
	}
	defer controllerConn.Close()

	err = controllerConn.DeleteTopics(topic.Name)
	if err != nil {
		panic(err.Error())
	}
}
