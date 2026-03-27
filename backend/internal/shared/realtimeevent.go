package shared

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type RTEventType string

const (
	RTEventTypeRich            RTEventType = "rich"
	RTEventTypeInvalidateCache RTEventType = "invalidate_cache"
	RTEventTypeHint            RTEventType = "hint"
	RTEventTypeCommandAck      RTEventType = "command_ack"
)

type RTEvent struct {
	MessageID    uuid.UUID            `json:"MessageID"`
	AggregateID  uuid.UUID            `json:"AggregateID"`
	ActionKey    ActionKey            `json:"ActionKey"`
	PartitionKey RealTimePartitionKey `json:"-"`
	Type         RTEventType          `json:"Type"`
	Payload      json.RawMessage      `json:"Payload"`
}

func (r RTEvent) Bytes() ([]byte, error)        { return json.Marshal(r) }
func (r RTEvent) GetPartitionKey() PartitionKey { return r.PartitionKey }
func (r RTEvent) GetActionKey() ActionKey       { return r.ActionKey }
func (r RTEvent) GetMessageID() uuid.UUID       { return r.MessageID }
func (r RTEvent) GetAggregateID() uuid.UUID     { return r.AggregateID }

func RTEventFromBytes(data []byte) (RTEvent, error) {
	var evt RTEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return RTEvent{}, fmt.Errorf("unmarshal rt event: %w", err)
	}
	return evt, nil
}

type WsAggregatedEvent struct {
	RTEvent      RTEvent                  `json:"RTEvent"`
	PartitionKey WsAggregatedPartitionKey `json:"-"`
}

func (w WsAggregatedEvent) Bytes() ([]byte, error)        { return json.Marshal(w) }
func (w WsAggregatedEvent) GetPartitionKey() PartitionKey { return w.PartitionKey }
func (w WsAggregatedEvent) GetActionKey() ActionKey       { return w.RTEvent.GetActionKey() }
func (w WsAggregatedEvent) GetMessageID() uuid.UUID       { return w.RTEvent.GetMessageID() }
func (w WsAggregatedEvent) GetAggregateID() uuid.UUID     { return w.RTEvent.GetAggregateID() }

func WsAggregatedEventFromBytes(data []byte) (WsAggregatedEvent, error) {
	var evt WsAggregatedEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return WsAggregatedEvent{}, fmt.Errorf("unmarshal ws aggregated event: %w", err)
	}
	return evt, nil
}
