package entities

import (
	"time"

	"github.com/google/uuid"
)

type Invite struct {
	ID          uuid.UUID `json:"Id"`
	WorkspaceID uuid.UUID `json:"WorkspaceId"`
	InvitedBy   uuid.UUID `json:"InvitedBy"`
	Token       string    `json:"Token"`
	Email       *string   `json:"Email,omitempty"`
	ExpiresAt   time.Time `json:"ExpiresAt"`
	MaxUses     int       `json:"MaxUses"`
	UseCount    int       `json:"UseCount"`
	Timestamps
}
