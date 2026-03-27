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
	WorkspaceID      uuid.UUID   `json:"WorkspaceID"`
	ChannelID        uuid.UUID   `json:"ChannelID"`
	AggregateID      uuid.UUID   `json:"AggregateID"`
	UserID           uuid.UUID   `json:"UserID"`
	MentionedUserIDs []uuid.UUID `json:"MentionedUserIDs,omitempty"`
	MentionChannel   bool        `json:"MentionChannel,omitempty"`
	MentionHere      bool        `json:"MentionHere,omitempty"`
}

type DMCreatedPayload struct {
	ChannelID      uuid.UUID   `json:"ChannelID"`
	WorkspaceID    uuid.UUID   `json:"WorkspaceID"`
	ParticipantIDs []uuid.UUID `json:"ParticipantIDs"`
}

type UserRegisteredPayload struct {
	UserID uuid.UUID `json:"UserID"`
}
