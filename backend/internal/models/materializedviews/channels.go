package materializedviews

import (
	"gokafka/internal/models/entities"
	"time"

	"github.com/google/uuid"
)

type ChannelMembershipView struct {
	ChannelID         uuid.UUID                  `gorm:"primaryKey" json:"ChannelId"`
	UserID            uuid.UUID                  `gorm:"primaryKey" json:"UserId"`
	WorkspaceID       uuid.UUID                  `json:"WorkspaceId"`
	Name              *string                    `json:"Name"`
	Description       *string                    `json:"Description"`
	Topic             *string                    `json:"Topic"`
	Type              entities.ChannelType       `json:"Type"`
	IsArchived        bool                       `json:"IsArchived"`
	CreatedBy         uuid.UUID                  `json:"CreatedBy"`
	Role              entities.ChannelMemberRole `json:"Role"`
	JoinedAt          time.Time                  `json:"JoinedAt"`
	NotificationsPref entities.NotificationPref  `json:"NotificationsPref"`
	LastReadAt        *time.Time                 `json:"LastReadAt"`
}

type ChannelView struct {
	ChannelID   uuid.UUID            `gorm:"primaryKey" json:"ChannelId"`
	WorkspaceID uuid.UUID            `json:"WorkspaceId"`
	CreatedBy   uuid.UUID            `json:"CreatedBy"`
	Name        *string              `json:"Name"`
	Description *string              `json:"Description"`
	Topic       *string              `json:"Topic"`
	Type        entities.ChannelType `json:"Type"`
	IsArchived  bool                 `json:"IsArchived"`
}

type DirectMessageMembershipView struct {
	ChannelID         uuid.UUID                 `gorm:"primaryKey" json:"ChannelId"`
	UserID            uuid.UUID                 `gorm:"primaryKey" json:"UserId"`
	WorkspaceID       uuid.UUID                 `json:"WorkspaceId"`
	JoinedAt          time.Time                 `json:"JoinedAt"`
	NotificationsPref entities.NotificationPref `json:"NotificationsPref"`
}
