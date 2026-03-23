package domaineventpayloads

import "github.com/google/uuid"

type WorkspaceCreatedPayload struct {
	WorkspaceID uuid.UUID `json:"WorkspaceID"`
	UserID      uuid.UUID `json:"UserID"`
}

type ChannelCreatedPayload struct {
	ChannelID   uuid.UUID `json:"ChannelID"`
	WorkspaceID uuid.UUID `json:"WorkspaceID"`
	UserID      uuid.UUID `json:"UserID"`
}
