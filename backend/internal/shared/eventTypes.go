package shared

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type KafkaDomainMessage interface {
	Bytes() []byte
	//FromBytes(data []byte) error
	GetPartitionKey() PartitionKey
	GetActionKey() ActionKey
}

type Event struct {
	EventID      uuid.UUID
	ActionKey    ActionKey
	PartitionKey EventPartitionKey
	Payload      interface{}
}

type EventJSON struct {
	EventID      uuid.UUID   `json:"EventID"`
	ActionKey    ActionKey   `json:"ActionKey"`
	PartitionKey string      `json:"PartitionKey"`
	Payload      interface{} `json:"Payload"`
}

func (e Event) Bytes() []byte {
	jsonEvent, err := MapEventToJSON(e)
	if err != nil {
		panic(err.Error())
	}
	eventJson, err := json.Marshal(jsonEvent)
	if err != nil {
		panic(err.Error())
	}
	return eventJson
}

func (e *Event) FromBytes(data []byte) error {
	var aux EventJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.EventID = aux.EventID
	e.ActionKey = aux.ActionKey

	pk, ok := ParsePartitionKey(aux.PartitionKey).(EventPartitionKey)
	if !ok {
		return fmt.Errorf("unexpected partition key")
	}
	e.PartitionKey = pk
	e.Payload = aux.Payload
	return nil
}

func (e Event) GetPartitionKey() PartitionKey {
	return e.PartitionKey
}

func (e Event) GetActionKey() ActionKey {
	return e.ActionKey
}

func NewEvent(actionKey ActionKey, partitionKey EventPartitionKey, payload interface{}) Event {
	return Event{
		EventID:      uuid.New(),
		ActionKey:    actionKey,
		PartitionKey: partitionKey,
		Payload:      payload,
	}
}

func MapEventToJSON(e Event) (EventJSON, error) {
	return EventJSON{
		EventID:      e.EventID,
		ActionKey:    e.ActionKey,
		PartitionKey: e.PartitionKey.String(),
		Payload:      e.Payload,
	}, nil
}

type Command struct {
	CommandID    uuid.UUID
	ActionKey    ActionKey
	PartitionKey CommandPartitionKey
	Payload      interface{}
}

type CommandJSON struct {
	CommandID    uuid.UUID   `json:"CommandID"`
	ActionKey    ActionKey   `json:"ActionKey"`
	PartitionKey string      `json:"PartitionKey"`
	Payload      interface{} `json:"Payload"`
}

func (c Command) Bytes() []byte {
	jsonCommand, err := MapCommandToJSON(c)
	if err != nil {
		panic(err.Error())
	}
	commandJson, err := json.Marshal(jsonCommand)
	if err != nil {
		panic(err.Error())
	}
	return commandJson
}

func (c *Command) FromBytes(data []byte) error {
	var aux CommandJSON
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.CommandID = aux.CommandID
	c.ActionKey = aux.ActionKey
	pk, ok := ParsePartitionKey(aux.PartitionKey).(CommandPartitionKey)
	if !ok {
		return fmt.Errorf("unexpected partition key")
	}
	c.PartitionKey = pk
	c.Payload = aux.Payload
	return nil
}

func MapCommandToJSON(c Command) (CommandJSON, error) {
	return CommandJSON{
		CommandID:    c.CommandID,
		ActionKey:    c.ActionKey,
		PartitionKey: c.PartitionKey.String(),
		Payload:      c.Payload,
	}, nil
}

func (c Command) GetPartitionKey() PartitionKey {
	return c.PartitionKey
}

func (c Command) GetActionKey() ActionKey {
	return c.ActionKey
}

func NewCommand(actionKey ActionKey, partitionKey CommandPartitionKey, payload interface{}) Command {
	return Command{
		CommandID:    uuid.New(),
		ActionKey:    actionKey,
		PartitionKey: partitionKey,
		Payload:      payload,
	}
}
