package api

type CreateWorkspaceRequest struct {
	Name    string  `json:"name" binding:"required"`
	IconURL *string `json:"iconUrl,omitempty"`
}

type CreateChannelRequest struct {
	Name              string  `json:"name" binding:"required"`
	Description       *string `json:"description,omitempty"`
	Topic             *string `json:"topic,omitempty"`
	Type              string  `json:"type" binding:"required,oneof=public private"`
	NotificationsPref *string `json:"notificationsPref" binding:"oneof=all mentions none"`
}

type CreateMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type ReplyToMessageRequest struct {
	Content string `json:"content" binding:"required"`
}
