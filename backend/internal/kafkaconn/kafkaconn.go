package kafkaconn

import "github.com/segmentio/kafka-go"

type KafkaConn struct {
	Conn    *kafka.Conn
	Brokers []string
}

func NewKafkaConn(address string, brokers []string) (*KafkaConn, error) {
	conn, err := kafka.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &KafkaConn{
		Conn:    conn,
		Brokers: brokers,
	}, nil
}

func (kc *KafkaConn) Close() error {
	return kc.Conn.Close()
}
