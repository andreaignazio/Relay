package shared

func NewWsClientTopic(clientID string, partitions int, replicas int) Topic {
	return Topic{
		Name:       "wsclient." + clientID,
		Partitions: partitions,
		Replicas:   replicas,
	}
}
