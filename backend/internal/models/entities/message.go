package entities

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID               uuid.UUID  `json:"Id"`
	ChannelID        uuid.UUID  `json:"ChannelId"`
	UserID           uuid.UUID  `json:"UserId"`
	ParentMessageID  *uuid.UUID `json:"ParentMessageId,omitempty"`
	Content          string     `json:"Content"`
	IsEdited         bool       `json:"IsEdited"`
	ReplyCount       int        `json:"ReplyCount"`
	LastReplyAt      *time.Time `json:"LastReplyAt,omitempty"`
	MentionedUserIDs UUIDArray  `json:"MentionedUserIds" gorm:"type:uuid[];not null;default:'{}'"`
	MentionChannel   bool       `json:"MentionChannel"`
	MentionHere      bool       `json:"MentionHere"`
	Timestamps
}

// Reaction — unique: (MessageID, UserID, Emoji)
type Reaction struct {
	ID        uuid.UUID `json:"Id"`
	MessageID uuid.UUID `json:"MessageId"`
	UserID    uuid.UUID `json:"UserId"`
	Emoji     string    `json:"Emoji"`
	Timestamps
}

type Attachment struct {
	ID        uuid.UUID `json:"Id"`
	MessageID uuid.UUID `json:"MessageId"`
	UserID    uuid.UUID `json:"UserId"`
	Filename  string    `json:"Filename"`
	FileURL   string    `json:"FileUrl"`
	MimeType  string    `json:"MimeType"`
	SizeBytes int64     `json:"SizeBytes"`
	Version   int       `json:"Version"`
	CreatedAt time.Time `json:"CreatedAt"`
}
