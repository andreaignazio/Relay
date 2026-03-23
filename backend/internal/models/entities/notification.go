package entities

import "github.com/google/uuid"

type Notification struct {
	ID            uuid.UUID        `json:"Id"`
	UserID        uuid.UUID        `json:"UserId"`
	WorkspaceID   uuid.UUID        `json:"WorkspaceId"`
	Type          NotificationType `json:"Type"`
	ReferenceID   uuid.UUID        `json:"ReferenceId"`
	ReferenceType string           `json:"ReferenceType"`
	IsRead        bool             `json:"IsRead"`
	Timestamps
}
