package shared

import (
	"fmt"
	"strings"
)

type RealTimeTopic string

var (
	RealTimeTopicWorkspaces RealTimeTopic = mustRealTimeTopic(RealTimeDestinationWorkspaces)
	RealTimeTopicChannels   RealTimeTopic = mustRealTimeTopic(RealTimeDestinationChannels)
	RealTimeTopicUsers      RealTimeTopic = mustRealTimeTopic(RealTimeDestinationUsers)
)

type RealTimeDestination string

const (
	RealTimeDestinationWorkspaces RealTimeDestination = "workspaces"
	RealTimeDestinationChannels   RealTimeDestination = "channels"
	RealTimeDestinationUsers      RealTimeDestination = "users"
)

func mustRealTimeTopic(rtd RealTimeDestination) RealTimeTopic {
	str := "realtime." + string(rtd)
	return RealTimeTopic(str)
}

func (rt RealTimeTopic) Topic(partitions int, replicas int) Topic {
	return Topic{
		Name:       string(rt),
		Partitions: partitions,
		Replicas:   replicas,
	}
}

func (rt RealTimeTopic) GetRealTimeDestination() (RealTimeDestination, error) {
	parts := strings.Split(string(rt), ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid real-time topic format: %s", rt)
	}
	return RealTimeDestination(parts[1]), nil
}
