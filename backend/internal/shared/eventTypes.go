package shared

import (
	"encoding/json"

	"github.com/google/uuid"
)

type KafkaWritable interface {
	Bytes() ([]byte, error)
	GetPartitionKey() PartitionKey
}

type KafkaDomainMessage interface {
	KafkaWritable
	GetActionKey() ActionKey
	GetAggregateID() uuid.UUID
	GetMessageID() uuid.UUID
}

type Event struct {
	MessageID     uuid.UUID                  `json:"MessageID"`
	AggregateID   uuid.UUID                  `json:"AggregateID"`
	ActionKey     ActionKey                  `json:"ActionKey"`
	AuthorID      uuid.UUID                  `json:"AuthorID"`
	PartitionKey  EventPartitionKey          `json:"-"`
	EntityContext map[EntityKeys][]uuid.UUID `json:"EntityContext,omitempty"`
	Payload       json.RawMessage            `json:"Payload,omitempty"`
}

func (e Event) Bytes() ([]byte, error) {
	return json.Marshal(e)
}

func EventFromBytes(data []byte) (Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return Event{}, err
	}
	return event, nil
}

func (e Event) GetPartitionKey() PartitionKey {
	return e.PartitionKey
}

func (e Event) GetActionKey() ActionKey {
	return e.ActionKey
}

func (e Event) GetAggregateID() uuid.UUID {
	return e.AggregateID
}

func (e Event) GetMessageID() uuid.UUID {
	return e.MessageID
}

func (e Event) GetAuthorID() uuid.UUID {
	return e.AuthorID
}

func NewEvent(
	messageID uuid.UUID,
	aggregateID uuid.UUID,
	actionKey ActionKey,
	partitionKey EventPartitionKey,
	entityContext map[EntityKeys][]uuid.UUID,
	payload json.RawMessage,
	authorID uuid.UUID,
) Event {
	return Event{
		MessageID:     messageID,
		AggregateID:   aggregateID,
		ActionKey:     actionKey,
		PartitionKey:  partitionKey,
		EntityContext: entityContext,
		Payload:       payload,
		AuthorID:      authorID,
	}
}

type Command struct {
	MessageID    uuid.UUID           `json:"MessageID"`
	AggregateID  uuid.UUID           `json:"AggregateID"`
	ActionKey    ActionKey           `json:"ActionKey"`
	AuthorID     uuid.UUID           `json:"AuthorID"`
	PartitionKey CommandPartitionKey `json:"-"`
	TraceID      uuid.UUID           `json:"TraceID"`
	Metadata     MessageMetadata     `json:"Metadata"`
	Payload      json.RawMessage     `json:"Payload"`
}

type MessageMetadata struct {
	UserID string         `json:"UserID"`
	Extra  map[string]any `json:"Extra,omitempty"`
}

func (c Command) Bytes() ([]byte, error) {
	return json.Marshal(c)
}

func CommandFromBytes(data []byte) (Command, error) {
	var cmd Command
	if err := json.Unmarshal(data, &cmd); err != nil {
		return Command{}, err
	}
	return cmd, nil
}

func (c Command) GetPartitionKey() PartitionKey {
	return c.PartitionKey
}

func (c Command) GetActionKey() ActionKey {
	return c.ActionKey
}

func (c Command) GetAggregateID() uuid.UUID {
	return c.AggregateID
}

func (c Command) GetMessageID() uuid.UUID {
	return c.MessageID
}

func (c Command) GetAuthorID() uuid.UUID {
	return c.AuthorID
}

func NewCommand(
	messageID uuid.UUID,
	aggregateID uuid.UUID,
	actionKey ActionKey,
	partitionKey CommandPartitionKey,
	traceID uuid.UUID,
	metadata MessageMetadata,
	payload json.RawMessage,
	authorID uuid.UUID,
) Command {
	return Command{
		MessageID:    messageID,
		AggregateID:  aggregateID,
		ActionKey:    actionKey,
		PartitionKey: partitionKey,
		TraceID:      traceID,
		Metadata:     metadata,
		Payload:      payload,
		AuthorID:     authorID,
	}
}
