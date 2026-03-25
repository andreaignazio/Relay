package entities

import (
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID          uuid.UUID `json:"Id"`
	Name        string    `json:"Name"`
	Slug        string    `json:"Slug"`
	IconURL     *string   `json:"IconUrl,omitempty"`
	Description *string   `json:"Description,omitempty"`
	Timestamps
}

// WorkspaceMembership — unique: (UserID, WorkspaceID)
type WorkspaceMembership struct {
	ID              uuid.UUID           `json:"Id"`
	UserID          uuid.UUID           `json:"UserId"`
	WorkspaceID     uuid.UUID           `json:"WorkspaceId"`
	Role            WorkspaceMemberRole `json:"Role"`
	StatusEmoji     *string             `json:"StatusEmoji,omitempty"`
	StatusText      *string             `json:"StatusText,omitempty"`
	StatusExpiresAt *time.Time          `json:"StatusExpiresAt,omitempty"`
	JoinedAt        time.Time           `json:"JoinedAt"`
	Timestamps
}
