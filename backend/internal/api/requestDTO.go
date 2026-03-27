package api

type CreateWorkspaceRequest struct {
	Name        string  `json:"name" binding:"required"`
	IconURL     *string `json:"iconUrl,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CreateChannelRequest struct {
	Name              string  `json:"name" binding:"required"`
	Description       *string `json:"description,omitempty"`
	Topic             *string `json:"topic,omitempty"`
	Type              string  `json:"type" binding:"required,oneof=public private"`
	NotificationsPref *string `json:"notificationsPref" binding:"oneof=all mentions none"`
}

type CreateDirectMessageRequest struct {
	RecipientIDs      []string `json:"recipientIds" binding:"required,min=1,dive,required"`
	NotificationsPref *string  `json:"notificationsPref" binding:"oneof=all mentions none"`
}

type CreateMessageRequest struct {
	Content          string   `json:"content" binding:"required"`
	MentionedUserIDs []string `json:"mentionedUserIds,omitempty"`
	MentionChannel   bool     `json:"mentionChannel,omitempty"`
	MentionHere      bool     `json:"mentionHere,omitempty"`
}

type ReplyToMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

type BatchUsersRequest struct {
	UserIds []string `json:"userIds" binding:"required,min=1,dive,required"`
}

type KratosRegistrationHookRequest struct {
	IdentityID  string `json:"identity_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
}
