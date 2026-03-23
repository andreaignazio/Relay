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
