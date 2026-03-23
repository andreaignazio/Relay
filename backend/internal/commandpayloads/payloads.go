package commandpayloads

import "github.com/google/uuid"

type CreateWorkspacePayload struct {
	Name    string  `json:"name"`
	IconURL *string `json:"iconUrl,omitempty"`
}

type CreateChannelPayload struct {
	WorkspaceID       uuid.UUID `json:"workspaceId"`
	Name              string    `json:"name"`
	Description       *string   `json:"description,omitempty"`
	Topic             *string   `json:"topic,omitempty"`
	Type              string    `json:"type"` // "public" or "private"
	NotificationsPref *string   `json:"notificationsPref,omitempty"`
}
