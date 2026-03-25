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

type CreateMessagePayload struct {
	WorkspaceID uuid.UUID `json:"workspaceId"`
	ChannelID   uuid.UUID `json:"channelId"`
	Content     string    `json:"content"`
}

type CreateDMPayload struct {
	WorkspaceID    uuid.UUID   `json:"workspaceId"`
	ParticipantIDs []uuid.UUID `json:"participantIds"`
}

type ReplyToMessagePayload struct {
	WorkspaceID     uuid.UUID `json:"workspaceId"`
	ChannelID       uuid.UUID `json:"channelId"`
	ParentMessageID uuid.UUID `json:"parentMessageId"`
	Content         string    `json:"content"`
}
