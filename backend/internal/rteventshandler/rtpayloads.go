package rteventshandler

import (
	"encoding/json"
	"gokafka/internal/shared"
	"time"

	"github.com/google/uuid"
)

// RTRichPayload — payload completo per update immediato della UI
type RTRichPayload struct {
	MessageID        uuid.UUID   `json:"MessageId"`
	ChannelID        uuid.UUID   `json:"ChannelId"`
	ParentMessageID  *uuid.UUID  `json:"ParentMessageId,omitempty"`
	Content          string      `json:"Content"`
	Author           RTAuthor    `json:"Author"`
	CreatedAt        time.Time   `json:"CreatedAt"`
	MentionedUserIDs []uuid.UUID `json:"MentionedUserIds,omitempty"`
	MentionChannel   bool        `json:"MentionChannel,omitempty"`
	MentionHere      bool        `json:"MentionHere,omitempty"`
}

type RTAuthor struct {
	UserID      uuid.UUID `json:"UserId"`
	DisplayName string    `json:"DisplayName"`
	AvatarURL   string    `json:"AvatarUrl"`
}

// RTInvalidationPayload — segnale di re-fetch
type RTInvalidationPayload struct {
	Resource string    `json:"Resource"`
	ID       uuid.UUID `json:"Id"`
}

// RTSidebarPayload — aggiorna contatori senza re-fetch
type RTSidebarPayload struct {
	ChannelID uuid.UUID `json:"ChannelId"`
	Hint      string    `json:"Hint"`
}

func NewRTRichEvent(
	messageID uuid.UUID,
	aggregateID uuid.UUID,
	actionKey shared.ActionKey,
	partitionKey shared.RealTimePartitionKey,
	payload json.RawMessage,
) shared.RTEvent {
	return shared.RTEvent{
		MessageID:    messageID,
		AggregateID:  aggregateID,
		ActionKey:    actionKey,
		Type:         shared.RTEventTypeRich,
		PartitionKey: partitionKey,
		Payload:      payload,
	}
}

func NewRTInvalidationEvent(
	messageID uuid.UUID,
	aggregateID uuid.UUID,
	actionKey shared.ActionKey,
	partitionKey shared.RealTimePartitionKey,
	resource string,
	resourceID uuid.UUID,
) shared.RTEvent {
	payload, _ := json.Marshal(RTInvalidationPayload{
		Resource: resource,
		ID:       resourceID,
	})
	return shared.RTEvent{
		MessageID:    messageID,
		AggregateID:  aggregateID,
		ActionKey:    actionKey,
		Type:         shared.RTEventTypeInvalidateCache,
		PartitionKey: partitionKey,
		Payload:      payload,
	}
}

func NewRTSidebarEvent(
	messageID uuid.UUID,
	aggregateID uuid.UUID,
	actionKey shared.ActionKey,
	partitionKey shared.RealTimePartitionKey,
	channelID uuid.UUID,
	hint string,
) shared.RTEvent {
	payload, _ := json.Marshal(RTSidebarPayload{
		ChannelID: channelID,
		Hint:      hint,
	})
	return shared.RTEvent{
		MessageID:    messageID,
		AggregateID:  aggregateID,
		ActionKey:    actionKey,
		Type:         shared.RTEventTypeHint,
		PartitionKey: partitionKey,
		Payload:      payload,
	}
}
