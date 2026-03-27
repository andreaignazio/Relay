package users

import (
	"context"
	"encoding/json"
	"fmt"
	"gokafka/internal/commandpayloads"
	"gokafka/internal/domaineventpayloads"
	"gokafka/internal/eventstorepayloads"
	"gokafka/internal/models"
	"gokafka/internal/models/entities"
	"gokafka/internal/shared"
	"time"

	"github.com/google/uuid"
)

func (s *Service) HandleUserRegisteredEvent(ctx context.Context, cmd shared.Command) error {
	s.LogChan <- fmt.Sprintf("[UserService] Handling UserRegister command for user ID: %s", cmd.AggregateID)

	var payload commandpayloads.RegisterUserPayload
	if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	userID := cmd.AggregateID
	messageID := cmd.GetMessageID()
	now := time.Now()

	userSnapshot := models.UserSnapshot{
		User: entities.User{
			ID:          userID,
			Email:       payload.Email,
			Username:    payload.Username,
			DisplayName: payload.DisplayName,
			Timestamps: entities.Timestamps{
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		SnapshotFields: models.SnapshotFields{
			SnapVersion:   1,
			SnapUpdatedAt: now,
		},
	}

	eventStorePayload := eventstorepayloads.UserRegisteredPayload{
		User: userSnapshot,
	}
	eventStorePayloadBytes, err := json.Marshal(eventStorePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event store payload: %w", err)
	}

	record := models.EventStore{
		EventID:       messageID.String(),
		AggregateID:   userID,
		AggregateType: shared.ActionKeyResourceUser,
		Version:       1,
		ActionKey:     shared.ActionKeyUserRegister,
		Payload:       eventStorePayloadBytes,
		Metadata: shared.MessageMetadata{
			UserID: userID.String(),
		},
		CreatedAt: now,
	}

	if err := s.tx.RunInTransaction(ctx, func(ctx context.Context) error {
		if err := s.EventStoreRepo.InsertEventTX(ctx, record); err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
		if err := s.SnapshotRepo.UpsertUserSnapshotTX(ctx, userSnapshot); err != nil {
			return fmt.Errorf("failed to upsert user snapshot: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	domainPayload := domaineventpayloads.UserRegisteredPayload{
		UserID: userID,
	}
	domainPayloadBytes, err := json.Marshal(domainPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal domain event payload: %w", err)
	}

	event := shared.NewEvent(
		messageID,
		userID,
		cmd.ActionKey,
		shared.NewUserEventPartitionKey(userID.String()),
		map[shared.EntityKeys][]uuid.UUID{
			shared.EntityKeysUser: {userID},
		},
		domainPayloadBytes,
		cmd.GetAuthorID(),
	)

	if err := s.ProduceEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to produce event: %w", err)
	}

	s.LogChan <- fmt.Sprintf("[UserService] Successfully registered user %s (%s)", payload.Username, userID)
	return nil
}
