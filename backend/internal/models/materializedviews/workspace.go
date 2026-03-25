package materializedviews

import (
	"gokafka/internal/models/entities"
	"time"

	"github.com/google/uuid"
)

type WorkspaceView struct {
	WorkspaceID uuid.UUID                    `gorm:"primaryKey" json:"WorkspaceId"`
	UserID      uuid.UUID                    `gorm:"primaryKey" json:"UserId"`
	Name        string                       `json:"Name"`
	Slug        string                       `json:"Slug"`
	IconURL     *string                      `json:"IconUrl,omitempty"`
	Description *string                      `json:"Description,omitempty"`
	CreatedAt   time.Time                    `json:"CreatedAt"`
	UpdatedAt   time.Time                    `json:"UpdatedAt"`
	Role        entities.WorkspaceMemberRole `json:"Role"`
	JoinedAt    time.Time                    `json:"JoinedAt"`
}
