package shared

import (
	"fmt"
	"strings"
)

type PartitionKeyKind string

const (
	PartitionKeyKindEvent        PartitionKeyKind = "evt"
	PartitionKeyKindCommand      PartitionKeyKind = "cmd"
	PartitionKeyKindRealTime     PartitionKeyKind = "rt"
	PartitionKeyKindWsCommand    PartitionKeyKind = "ws"
	PartitionKeyKindWsAggregated PartitionKeyKind = "wsa"
)

type PartitionKey interface {
	Bytes() []byte
	String() string
	ID() string
}

type EventPartitionKey struct {
	Kind         PartitionKeyKind
	ParentEntity EntityKeys
	ParentID     string
}

type CommandPartitionKey struct {
	Kind         PartitionKeyKind
	ServicesKeys ServicesKeys
	AggregateID  string
}

type RealTimePartitionKey struct {
	Kind                PartitionKeyKind
	RealTimeDestination RealTimeDestination
	EntityID            string
}

type WsCommandPartitionKey struct {
	Kind     PartitionKeyKind
	Constant string
	ClientID string
}

type WsAggregatedPartitionKey struct {
	Kind     PartitionKeyKind
	Constant string
	ClientID string
}

func (epk EventPartitionKey) String() string {
	return string(epk.Kind) + ":" + string(epk.ParentEntity) + ":" + epk.ParentID
}

func (epk EventPartitionKey) Bytes() []byte {
	return []byte(epk.String())
}

func (epk EventPartitionKey) ID() string {
	return epk.ParentID
}

func (cpk CommandPartitionKey) String() string {
	return string(cpk.Kind) + ":" + string(cpk.ServicesKeys) + ":" + cpk.AggregateID
}

func (cpk CommandPartitionKey) Bytes() []byte {
	return []byte(cpk.String())
}
func (cpk CommandPartitionKey) ID() string {
	return cpk.AggregateID
}

func (rtpk RealTimePartitionKey) String() string {
	return string(rtpk.Kind) + ":" + string(rtpk.RealTimeDestination) + ":" + rtpk.EntityID
}

func (rtpk RealTimePartitionKey) Bytes() []byte {
	return []byte(rtpk.String())
}

func (rtpk RealTimePartitionKey) ID() string {
	return rtpk.EntityID
}

func (rtpk RealTimePartitionKey) GetRealTimeDestination() RealTimeDestination {
	return rtpk.RealTimeDestination
}

func (wapk WsAggregatedPartitionKey) String() string {
	return string(wapk.Kind) + ":" + wapk.Constant + ":" + wapk.ClientID
}

func (wapk WsAggregatedPartitionKey) Bytes() []byte {
	return []byte(wapk.String())
}

func (wapk WsAggregatedPartitionKey) ID() string {
	return wapk.ClientID
}

func (wspk WsCommandPartitionKey) String() string {
	return string(wspk.Kind) + ":" + wspk.Constant + ":" + wspk.ClientID
}

func (wspk WsCommandPartitionKey) Bytes() []byte {
	return []byte(wspk.String())
}

func (wspk WsCommandPartitionKey) ID() string {
	return wspk.ClientID
}

func parseWsCommandPartitionKey(parts []string) WsCommandPartitionKey {
	constant := parts[1]
	clientID := parts[2]
	return WsCommandPartitionKey{
		Kind:     PartitionKeyKindWsCommand,
		Constant: constant,
		ClientID: clientID,
	}
}

func ParsePartitionKey(keyStr string) (PartitionKey, error) {
	parts := strings.Split(keyStr, ":")
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid partition key format: %s", keyStr)
	}
	switch PartitionKeyKind(parts[0]) {
	case PartitionKeyKindEvent:
		return parseEventPartitionKey(parts), nil
	case PartitionKeyKindCommand:
		return parseCommandPartitionKey(parts), nil
	case PartitionKeyKindRealTime:
		return parseRealTimePartitionKey(parts), nil
	case PartitionKeyKindWsCommand:
		return parseWsCommandPartitionKey(parts), nil
	case PartitionKeyKindWsAggregated:
		return parseWsAggregatedPartitionKey(parts), nil
	default:
		return nil, fmt.Errorf("unknown partition key kind: %s", parts[0])
	}
}

func parseEventPartitionKey(parts []string) EventPartitionKey {

	prefix, id := parts[1], parts[2]
	switch EntityKeys(prefix) {
	case EntityKeysWorkspace:
		return NewWorkspaceEventPartitionKey(id)
	default:
		panic("unknown partition key prefix: " + prefix)
	}
}

func parseCommandPartitionKey(parts []string) PartitionKey {

	prefix, id := parts[1], parts[2]
	switch ServicesKeys(prefix) {
	case ServicesKeysWorkspaces:
		return NewWorkspaceCommandPartitionKey(id)
	case ServicesKeysChannels:
		return NewChannelCommandPartitionKey(id)

	default:
		panic("unknown command partition key prefix: " + prefix)
	}
}

func parseRealTimePartitionKey(parts []string) PartitionKey {
	prefix, id := parts[1], parts[2]
	switch RealTimeDestination(prefix) {
	case RealTimeDestinationWorkspaces:
		return NewWorkspaceRealTimePartitionKey(id)
	case RealTimeDestinationChannels:
		return NewChannelRealTimePartitionKey(id)
	case RealTimeDestinationUsers:
		return NewUserRealTimePartitionKey(id)
	default:
		panic("unknown real-time partition key prefix: " + prefix)
	}
}

func parseWsAggregatedPartitionKey(parts []string) WsAggregatedPartitionKey {
	constant := parts[1]
	clientID := parts[2]
	return WsAggregatedPartitionKey{
		Kind:     PartitionKeyKindWsAggregated,
		Constant: constant,
		ClientID: clientID,
	}
}

func NewWorkspaceEventPartitionKey(workspaceID string) EventPartitionKey {
	return EventPartitionKey{
		Kind:         PartitionKeyKindEvent,
		ParentEntity: EntityKeysWorkspace,
		ParentID:     workspaceID,
	}
}

func NewWorkspaceCommandPartitionKey(workspaceID string) CommandPartitionKey {
	return CommandPartitionKey{
		Kind:         PartitionKeyKindCommand,
		ServicesKeys: ServicesKeysWorkspaces,
		AggregateID:  workspaceID,
	}
}

func NewChannelCommandPartitionKey(channelID string) CommandPartitionKey {
	return CommandPartitionKey{
		Kind:         PartitionKeyKindCommand,
		ServicesKeys: ServicesKeysChannels,
		AggregateID:  channelID,
	}
}

func NewMessageCommandPartitionKey(channelID string) CommandPartitionKey {
	return CommandPartitionKey{
		Kind:         PartitionKeyKindCommand,
		ServicesKeys: ServicesKeysMessages,
		AggregateID:  channelID,
	}
}

func NewChannelEventPartitionKey(workspaceID string) EventPartitionKey {
	return NewWorkspaceEventPartitionKey(workspaceID)
}

func NewMessageEventPartitionKey(channelID string) EventPartitionKey {
	return EventPartitionKey{
		Kind:         PartitionKeyKindEvent,
		ParentEntity: EntityKeysChannel,
		ParentID:     channelID,
	}
}

func NewWorkspaceRealTimePartitionKey(workspaceID string) RealTimePartitionKey {
	return RealTimePartitionKey{
		Kind:                PartitionKeyKindRealTime,
		RealTimeDestination: RealTimeDestinationWorkspaces,
		EntityID:            workspaceID,
	}
}

func NewChannelRealTimePartitionKey(channelID string) RealTimePartitionKey {
	return RealTimePartitionKey{
		Kind:                PartitionKeyKindRealTime,
		RealTimeDestination: RealTimeDestinationChannels,
		EntityID:            channelID,
	}
}

func NewUserRealTimePartitionKey(userID string) RealTimePartitionKey {
	return RealTimePartitionKey{
		Kind:                PartitionKeyKindRealTime,
		RealTimeDestination: RealTimeDestinationUsers,
		EntityID:            userID,
	}
}

func NewWsCommandPartitionKey(clientID string) WsCommandPartitionKey {
	return WsCommandPartitionKey{
		Kind:     PartitionKeyKindWsCommand,
		ClientID: clientID,
		Constant: "client",
	}
}

func NewWsAggregatedPartitionKey(clientID string) WsAggregatedPartitionKey {
	return WsAggregatedPartitionKey{
		Kind:     PartitionKeyKindWsAggregated,
		ClientID: clientID,
		Constant: "client",
	}
}
