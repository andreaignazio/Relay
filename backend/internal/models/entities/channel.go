package entities

import (
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID          uuid.UUID   `json:"Id"`
	WorkspaceID uuid.UUID   `json:"WorkspaceId"`
	CreatedBy   uuid.UUID   `json:"CreatedBy"`
	Name        *string     `json:"Name,omitempty"`
	Description *string     `json:"Description,omitempty"`
	Topic       *string     `json:"Topic,omitempty"`
	Type        ChannelType `json:"Type"`
	IsArchived  bool        `json:"IsArchived"`
	Timestamps
}

// ChannelMembership — unique: (UserID, ChannelID)
type ChannelMembership struct {
	ID                uuid.UUID         `json:"Id"`
	UserID            uuid.UUID         `json:"UserId"`
	ChannelID         uuid.UUID         `json:"ChannelId"`
	Role              ChannelMemberRole `json:"Role"`
	NotificationsPref NotificationPref  `json:"NotificationsPref"`
	LastReadAt        *time.Time        `json:"LastReadAt,omitempty"`
	JoinedAt          time.Time         `json:"JoinedAt"`
	Timestamps
}
