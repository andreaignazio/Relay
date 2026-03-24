package messages

import (
	"context"
	"encoding/json"
	"gokafka/internal/commandpayloads"
	"gokafka/internal/domaineventpayloads"
	"gokafka/internal/eventstorepayloads"
	"gokafka/internal/models"
	"gokafka/internal/models/entities"
	"gokafka/internal/shared"
	"time"

	"github.com/google/uuid"
)

func (s *Service) HandleCreateMessage(ctx context.Context, cmd shared.Command) error {
	// Unmarshal payload
	var payload commandpayloads.CreateMessagePayload
	if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
		return err
	}

	userID, err := uuid.Parse(cmd.Metadata.UserID)
	if err != nil {
		return err
	}
	message := entities.Message{
		ID:         cmd.AggregateID,
		ChannelID:  payload.ChannelID,
		UserID:     userID,
		Content:    payload.Content,
		IsEdited:   false,
		ReplyCount: 0,
		Timestamps: entities.Timestamps{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	messageSnapshot := models.MessageSnapshot{
		Message: message,
		SnapshotFields: models.SnapshotFields{
			SnapVersion:   1,
			SnapUpdatedAt: time.Now(),
		},
	}

	storedPayload := eventstorepayloads.MessageCreatedPayload{
		Message: messageSnapshot,
	}
	payloadBytes, err := json.Marshal(storedPayload)
	if err != nil {
		return err
	}
	event := models.EventStore{
		EventID:       cmd.MessageID.String(),
		AggregateID:   cmd.AggregateID,
		AggregateType: shared.ActionKeyResourceMessage,
		Payload:       payloadBytes,
		Version:       1,
		ActionKey:     shared.ActionKeyMessageSend,
		Metadata:      cmd.Metadata,
		CreatedAt:     time.Now(),
	}
	if err := s.tx.RunInTransaction(ctx, func(ctx context.Context) error {
		// Insert event into EventStore
		if err := s.EventStoreRepo.InsertEventTX(ctx, event); err != nil {
			return err
		}
		// Update snapshot in SnapshotRepo
		if err := s.SnapshotRepo.UpsertMessageSnapshotTX(ctx, messageSnapshot); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	domainPayload := domaineventpayloads.MessageCreatedPayload{
		WorkspaceID: payload.WorkspaceID,
		ChannelID:   payload.ChannelID,
		AggregateID: cmd.AggregateID,
		UserID:      userID,
	}
	domainPayloadBytes, err := json.Marshal(domainPayload)
	if err != nil {
		return err
	}
	domainEvent := shared.NewEvent(
		cmd.MessageID,
		cmd.AggregateID,
		shared.ActionKeyMessageSend,
		shared.NewMessageEventPartitionKey(payload.ChannelID.String()),
		map[shared.EntityKeys]uuid.UUID{
			shared.EntityKeysWorkspace: payload.WorkspaceID,
			shared.EntityKeysChannel:   payload.ChannelID,
			shared.EntityKeysUser:      userID,
		},
		domainPayloadBytes)
	if err := s.ProduceEvent(ctx, domainEvent); err != nil {
		return err
	}

	return nil

}
