package channels

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

func (s *Service) HandleCreateChannel(ctx context.Context, cmd shared.Command) error {
	s.LogChan <- fmt.Sprintf("[Service] Handling CreateChannel command for channel ID: %s", cmd.GetAggregateID())

	// Create the event
	channelID := cmd.GetAggregateID()

	var payload commandpayloads.CreateChannelPayload
	if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	workspaceID := payload.WorkspaceID
	userID, err := uuid.Parse(cmd.Metadata.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	messageID := cmd.GetMessageID()

	newChannel := entities.Channel{
		ID:          channelID,
		WorkspaceID: workspaceID,
		CreatedBy:   userID,
		Name:        &payload.Name,
		Description: payload.Description,
		Topic:       payload.Topic,
		Type:        entities.ChannelType(payload.Type),
		IsArchived:  false,
	}
	var notificationPref entities.NotificationPref
	if payload.NotificationsPref != nil {
		notificationPref = entities.NotificationPref(*payload.NotificationsPref)
	} else {
		notificationPref = entities.NotificationPrefAll // Default preference
	}

	newChannelMembership := entities.ChannelMembership{
		ID:                uuid.New(),
		UserID:            userID,
		ChannelID:         channelID,
		Role:              entities.ChannelMemberRoleAdmin,
		NotificationsPref: notificationPref,
		JoinedAt:          time.Now(),
		Timestamps: entities.Timestamps{
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	channelSnapshot := models.ChannelSnapshot{
		Channel: newChannel,
		SnapshotFields: models.SnapshotFields{
			SnapVersion:   1,
			SnapUpdatedAt: time.Now(),
		},
	}

	membershipSnapshot := models.ChannelMembershipSnapshot{
		ChannelMembership: newChannelMembership,
		SnapshotFields: models.SnapshotFields{
			SnapVersion:   1,
			SnapUpdatedAt: time.Now(),
		},
	}

	storedEventPayload := eventstorepayloads.ChannelCreatedPayload{
		Channel:    channelSnapshot,
		Membership: membershipSnapshot,
	}
	storedEventPayloadJson, err := json.Marshal(storedEventPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	storedEvent := models.EventStore{
		EventID:       messageID.String(),
		Payload:       storedEventPayloadJson,
		AggregateID:   channelID,
		AggregateType: shared.ActionKeyResourceChannel,
		ActionKey:     shared.ActionKeyChannelCreate,
		Version:       1,
		Metadata:      cmd.Metadata,
		CreatedAt:     time.Now(),
	}

	if err := s.tx.RunInTransaction(ctx, func(ctx context.Context) error {
		// Insert event into EventStore
		if err := s.EventStoreRepo.InsertEventTX(ctx, storedEvent); err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
		// Upsert snapshots
		if err := s.SnapshotRepo.UpsertChannelSnapshotTX(ctx, channelSnapshot); err != nil {
			return fmt.Errorf("failed to upsert channel snapshot: %w", err)
		}
		if err := s.SnapshotRepo.UpsertChannelMembershipSnapshotTX(ctx, membershipSnapshot); err != nil {
			return fmt.Errorf("failed to upsert channel membership snapshot: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	domainPayload := domaineventpayloads.ChannelCreatedPayload{
		ChannelID:   channelID,
		WorkspaceID: workspaceID,
		UserID:      userID,
		Type:        entities.ChannelType(payload.Type),
	}
	domainPayloadBytes, err := json.Marshal(domainPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal domain event payload: %w", err)
	}

	event := shared.NewEvent(
		messageID,
		workspaceID,
		cmd.ActionKey,
		shared.NewWorkspaceEventPartitionKey(workspaceID.String()),
		domainPayloadBytes,
	)

	if err := s.ProduceEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to produce event: %w", err)
	}
	s.LogChan <- fmt.Sprintf("[Service] Successfully handled CreateChannel command for channel ID: %s with Name: %s", cmd.GetAggregateID(), payload.Name)
	return nil

}
