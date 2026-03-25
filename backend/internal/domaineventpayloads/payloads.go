package domaineventpayloads

import (
	"gokafka/internal/models/entities"

	"github.com/google/uuid"
)

type WorkspaceCreatedPayload struct {
	WorkspaceID uuid.UUID `json:"WorkspaceID"`
	UserID      uuid.UUID `json:"UserID"`
}

type ChannelCreatedPayload struct {
	ChannelID   uuid.UUID            `json:"ChannelID"`
	WorkspaceID uuid.UUID            `json:"WorkspaceID"`
	UserID      uuid.UUID            `json:"UserID"`
	Type        entities.ChannelType `json:"Type"`
}

type MessageCreatedPayload struct {
	WorkspaceID uuid.UUID `json:"WorkspaceID"`
	ChannelID   uuid.UUID `json:"ChannelID"`
	AggregateID uuid.UUID `json:"AggregateID"`
	UserID      uuid.UUID `json:"UserID"`
}

type DMCreatedPayload struct {
	ChannelID      uuid.UUID   `json:"ChannelID"`
	WorkspaceID    uuid.UUID   `json:"WorkspaceID"`
	ParticipantIDs []uuid.UUID `json:"ParticipantIDs"`
}
