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

// --- helpers ---

func buildChannelSnapshot(channel entities.Channel) models.ChannelSnapshot {
	return models.ChannelSnapshot{
		Channel: channel,
		SnapshotFields: models.SnapshotFields{
			SnapVersion:   1,
			SnapUpdatedAt: time.Now(),
		},
	}
}

func buildMembershipSnapshot(membership entities.ChannelMembership) models.ChannelMembershipSnapshot {
	return models.ChannelMembershipSnapshot{
		ChannelMembership: membership,
		SnapshotFields: models.SnapshotFields{
			SnapVersion:   1,
			SnapUpdatedAt: time.Now(),
		},
	}
}

func (s *Service) persistChannelCreation(ctx context.Context, storedEvent models.EventStore, channelSnapshot models.ChannelSnapshot, membershipSnapshots []models.ChannelMembershipSnapshot) error {
	return s.tx.RunInTransaction(ctx, func(ctx context.Context) error {
		if err := s.EventStoreRepo.InsertEventTX(ctx, storedEvent); err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
		if err := s.SnapshotRepo.UpsertChannelSnapshotTX(ctx, channelSnapshot); err != nil {
			return fmt.Errorf("failed to upsert channel snapshot: %w", err)
		}
		for _, ms := range membershipSnapshots {
			if err := s.SnapshotRepo.UpsertChannelMembershipSnapshotTX(ctx, ms); err != nil {
				return fmt.Errorf("failed to upsert membership snapshot: %w", err)
			}
		}
		return nil
	})
}

// --- handlers ---

func (s *Service) HandleCreateChannel(ctx context.Context, cmd shared.Command) error {
	s.LogChan <- fmt.Sprintf("[Service] Handling CreateChannel command for channel ID: %s", cmd.GetAggregateID())

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
		Timestamps:  entities.Timestamps{CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	var notificationPref entities.NotificationPref
	if payload.NotificationsPref != nil {
		notificationPref = entities.NotificationPref(*payload.NotificationsPref)
	} else {
		notificationPref = entities.NotificationPrefAll
	}

	newChannelMembership := entities.ChannelMembership{
		ID:                uuid.New(),
		UserID:            userID,
		ChannelID:         channelID,
		Role:              entities.ChannelMemberRoleAdmin,
		NotificationsPref: notificationPref,
		JoinedAt:          time.Now(),
		Timestamps:        entities.Timestamps{CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	channelSnapshot := buildChannelSnapshot(newChannel)
	membershipSnapshot := buildMembershipSnapshot(newChannelMembership)

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

	if err := s.persistChannelCreation(ctx, storedEvent, channelSnapshot, []models.ChannelMembershipSnapshot{membershipSnapshot}); err != nil {
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
		map[shared.EntityKeys][]uuid.UUID{
			shared.EntityKeysWorkspace: {workspaceID},
			shared.EntityKeysChannel:   {channelID},
			shared.EntityKeysUser:      {userID},
		},
		domainPayloadBytes,
	)

	if err := s.ProduceEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to produce event: %w", err)
	}
	s.LogChan <- fmt.Sprintf("[Service] Successfully handled CreateChannel command for channel ID: %s with Name: %s", cmd.GetAggregateID(), payload.Name)
	return nil
}

func (s *Service) HandleCreateDM(ctx context.Context, cmd shared.Command) error {
	s.LogChan <- fmt.Sprintf("[Service] Handling CreateDM command for channel ID: %s", cmd.GetAggregateID())

	channelID := cmd.GetAggregateID()
	messageID := cmd.GetMessageID()

	var payload commandpayloads.CreateDMPayload
	if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	workspaceID := payload.WorkspaceID
	userID, err := uuid.Parse(cmd.Metadata.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	existingID, found, err := s.SnapshotRepo.FindExistingDMChannel(ctx, workspaceID, payload.ParticipantIDs)
	if err != nil {
		return fmt.Errorf("failed to check existing DM channel: %w", err)
	}
	if found {
		s.LogChan <- fmt.Sprintf("[Service] DM channel already exists: %s — skipping creation", existingID)
		return nil
	}

	newChannel := entities.Channel{
		ID:          channelID,
		WorkspaceID: workspaceID,
		CreatedBy:   userID,
		Type:        entities.ChannelTypeDM,
		IsArchived:  false,
		Timestamps:  entities.Timestamps{CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	channelSnapshot := buildChannelSnapshot(newChannel)

	membershipSnapshots := make([]models.ChannelMembershipSnapshot, len(payload.ParticipantIDs))
	for i, participantID := range payload.ParticipantIDs {
		membership := entities.ChannelMembership{
			ID:                uuid.New(),
			UserID:            participantID,
			ChannelID:         channelID,
			Role:              entities.ChannelMemberRoleMember,
			NotificationsPref: entities.NotificationPrefAll,
			JoinedAt:          time.Now(),
			Timestamps:        entities.Timestamps{CreatedAt: time.Now(), UpdatedAt: time.Now()},
		}
		membershipSnapshots[i] = buildMembershipSnapshot(membership)
	}

	storedPayload := eventstorepayloads.DMCreatedPayload{
		Channel:     channelSnapshot,
		Memberships: membershipSnapshots,
	}
	storedPayloadBytes, err := json.Marshal(storedPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal stored payload: %w", err)
	}

	storedEvent := models.EventStore{
		EventID:       messageID.String(),
		AggregateID:   channelID,
		AggregateType: shared.ActionKeyResourceDM,
		ActionKey:     shared.ActionKeyDMCreate,
		Version:       1,
		Metadata:      cmd.Metadata,
		Payload:       storedPayloadBytes,
		CreatedAt:     time.Now(),
	}

	if err := s.persistChannelCreation(ctx, storedEvent, channelSnapshot, membershipSnapshots); err != nil {
		return err
	}

	domainPayload := domaineventpayloads.DMCreatedPayload{
		ChannelID:      channelID,
		WorkspaceID:    workspaceID,
		ParticipantIDs: payload.ParticipantIDs,
	}
	domainPayloadBytes, err := json.Marshal(domainPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal domain payload: %w", err)
	}

	event := shared.NewEvent(
		messageID,
		channelID,
		shared.ActionKeyDMCreate,
		shared.NewWorkspaceEventPartitionKey(workspaceID.String()),
		map[shared.EntityKeys][]uuid.UUID{
			shared.EntityKeysWorkspace: {workspaceID},
			shared.EntityKeysDM:        {channelID},
			shared.EntityKeysUser:      payload.ParticipantIDs,
		},
		domainPayloadBytes,
	)

	if err := s.ProduceEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to produce event: %w", err)
	}
	s.LogChan <- fmt.Sprintf("[Service] Successfully handled CreateDM command for channel ID: %s", channelID)
	return nil
}
