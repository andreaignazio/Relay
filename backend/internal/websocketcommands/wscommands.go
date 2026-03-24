package websocketcommands

import (
	"encoding/json"
	"gokafka/internal/shared"

	"github.com/google/uuid"
)

// WsCommandAck is the typed confirmation sent from Aggregator → Hub → Client
// after a WsCommand is processed. The frontend uses it to unblock the UI.
type WsCommandAck struct {
	WsCommandKey WsCommandKey `json:"WsCommandKey"`
	WsCommandID  uuid.UUID    `json:"WsCommandID"`
	Success      bool         `json:"Success"`
	Error        string       `json:"Error,omitempty"`
}

func (a WsCommandAck) Bytes() ([]byte, error) { return json.Marshal(a) }

// AggregatorAck is the internal in-process message sent from Aggregator to Hub.
type AggregatorAck struct {
	ClientID uuid.UUID
	Ack      WsCommandAck
}

type WsCommandMessage struct {
	WsCommandKey WsCommandKey        `json:"WsCommandKey"`
	WsCommandID  uuid.UUID           `json:"WsCommandID"`
	PartitionKey shared.PartitionKey `json:"-"`
	Payload      json.RawMessage     `json:"Payload"`
}

type WsCommandKey string

const (
	WsCommandKeyConnect     WsCommandKey = "Connect"
	WsCommandKeySubscribe   WsCommandKey = "Subscribe"
	WsCommandKeyUnsubscribe WsCommandKey = "Unsubscribe"
	WsCommandKeyDisconnect  WsCommandKey = "Disconnect"
	WsCommandKeyFocus       WsCommandKey = "Focus"
)

func (s WsCommandMessage) Bytes() ([]byte, error) {
	return json.Marshal(s)
}

func (s WsCommandMessage) GetPartitionKey() shared.PartitionKey {
	return s.PartitionKey
}

func WsCommandMessageFromBytes(data []byte) (*WsCommandMessage, error) {
	var cmd WsCommandMessage
	if err := json.Unmarshal(data, &cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

func NewWsCommandMessage(cmdKey WsCommandKey, wsCommandId uuid.UUID, clientID string,
	payload json.RawMessage) WsCommandMessage {
	partitionKey := shared.NewWsCommandPartitionKey(clientID)
	return WsCommandMessage{
		WsCommandKey: cmdKey,
		WsCommandID:  wsCommandId,
		PartitionKey: partitionKey,
		Payload:      payload,
	}
}
