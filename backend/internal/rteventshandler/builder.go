package rteventshandler

import (
	"context"
	"encoding/json"
	"fmt"
	"gokafka/internal/models"
	"gokafka/internal/shared"

	"github.com/google/uuid"
)

type SnapshotRepo interface {
	GetMessageSnapshot(ctx context.Context, aggregateID uuid.UUID, snapshot *models.MessageSnapshot) error
	GetUserSnapshot(ctx context.Context, userID uuid.UUID, snapshot *models.UserSnapshot) error
}

func (h *Handler) buildPayload(ctx context.Context, event shared.Event, eventType shared.RTEventType) (json.RawMessage, error) {
	switch eventType {
	case shared.RTEventTypeInvalidateCache:
		return h.buildInvalidateCachePayload(event)
	case shared.RTEventTypeRich:
		return h.buildRichPayload(ctx, event)
	case shared.RTEventTypeHint:
		return h.buildHintPayload(event)
	default:
		return nil, fmt.Errorf("unknown RTEventType: %s", eventType)
	}
}

func (h *Handler) buildInvalidateCachePayload(event shared.Event) (json.RawMessage, error) {
	return event.Payload, nil
}

func (h *Handler) buildRichPayload(ctx context.Context, event shared.Event) (json.RawMessage, error) {
	switch event.GetActionKey() {
	case shared.ActionKeyMessageSend, shared.ActionKeyMessageEdit, shared.ActionKeyMessageDelete:
		return h.buildMessageRichPayload(ctx, event)
	default:
		return nil, fmt.Errorf("no rich payload builder for ActionKey: %s", event.GetActionKey())
	}
}

func (h *Handler) buildMessageRichPayload(ctx context.Context, event shared.Event) (json.RawMessage, error) {
	var msg models.MessageSnapshot
	if err := h.SnapshotRepo.GetMessageSnapshot(ctx, event.GetAggregateID(), &msg); err != nil {
		return nil, fmt.Errorf("get message snapshot: %w", err)
	}

	var user models.UserSnapshot
	if err := h.SnapshotRepo.GetUserSnapshot(ctx, msg.UserID, &user); err != nil {
		return nil, fmt.Errorf("get user snapshot: %w", err)
	}

	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	payload := RTRichPayload{
		MessageID:       msg.ID,
		ChannelID:       msg.ChannelID,
		ParentMessageID: msg.ParentMessageID,
		Content:         msg.Content,
		CreatedAt:       msg.CreatedAt,
		Author: RTAuthor{
			UserID:      user.ID,
			DisplayName: user.DisplayName,
			AvatarURL:   avatarURL,
		},
	}
	return json.Marshal(payload)
}

func (h *Handler) buildHintPayload(event shared.Event) (json.RawMessage, error) {
	channelID, ok := event.EntityContext[shared.EntityKeysChannel]
	if !ok {
		return nil, nil
	}
	payload := RTSidebarPayload{
		ChannelID: channelID,
		Hint:      "increment",
	}
	return json.Marshal(payload)
}
