package websocketcommands

import (
	"encoding/json"

	"github.com/google/uuid"
)

// FrontendWsMessage is the raw message received from the frontend via WebSocket.
// It mirrors the TypeScript WsCommandMessage type in src/types/wscommands.ts.
// ClientID and UserID are not present — ReadPump enriches the message before
// producing to subscriptions.commands.
type FrontendWsMessage struct {
	WsCommandKey WsCommandKey    `json:"WsCommandKey"`
	Payload      json.RawMessage `json:"Payload"`
}

// FrontendSubscribePayload mirrors TypeScript SubscribePayload.
type FrontendSubscribePayload struct {
	Subscriptions []Subscription `json:"Subscriptions"`
}

// FrontendUnsubscribePayload mirrors TypeScript UnsubscribePayload.
type FrontendUnsubscribePayload struct {
	Subscriptions []Subscription `json:"Subscriptions"`
}

// FrontendFocusPayload mirrors TypeScript FocusPayload.
type FrontendFocusPayload struct {
	ChannelID uuid.UUID `json:"ChannelID"`
}
