package websocketcommands

import (
	"gokafka/internal/shared"

	"github.com/google/uuid"
)

type SubscribeCommandPayload struct {
	UserID        uuid.UUID      `json:"UserID"`
	ClientID      uuid.UUID      `json:"ClientID"`
	Subscriptions []Subscription `json:"Subscriptions"`
}

type UnsubscribeCommandPayload struct {
	UserID        uuid.UUID      `json:"UserID"`
	ClientID      uuid.UUID      `json:"ClientID"`
	Subscriptions []Subscription `json:"Subscriptions"`
}

type DisconnectCommandPayload struct {
	//UserID   uuid.UUID `json:"UserID"`
	ClientID uuid.UUID `json:"ClientID"`
}

type FocusCommandPayload struct {
	UserID    uuid.UUID `json:"UserID"`
	ClientID  uuid.UUID `json:"ClientID"`
	ChannelID uuid.UUID `json:"ChannelID"`
}

type ConnectCommandPayload struct {
	UserID   uuid.UUID `json:"UserID"`
	ClientID uuid.UUID `json:"ClientID"`
}

type Subscription struct {
	RealTimeTopic shared.RealTimeTopic `json:"RealTimeTopic"`
	ID            uuid.UUID            `json:"ID"`
}
