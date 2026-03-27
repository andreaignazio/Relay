package workspaces

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

func (s *Service) HandleCreateWorkspace(ctx context.Context, command shared.Command) error {
	//fmt.Println("[Service] Handling CreateWorkspace command for workspace ID:", event.GetPartitionKey())
	s.LogChan <- fmt.Sprintf("[Service] Handling CreateWorkspace command for workspace ID: %s", command.GetAggregateID())
	workspaceID := command.GetAggregateID()
	userID, err := uuid.Parse(command.Metadata.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %s", command.Metadata.UserID)
	}
	messageID := command.GetMessageID()

	var payload commandpayloads.CreateWorkspacePayload
	if err := json.Unmarshal(command.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	slug := s.GenerateSlug(ctx, string(payload.Name))

	//Read workspaceSnapshot for workspaceID to get current state (if any)
	var workspaceSnapshot models.WorkspaceSnapshot
	/*if err := s.SnapshotRepo.GetSnapshot(ctx, workspaceID, &snapshot); err != nil {
		// If snapshot not found, we can proceed with an empty state
	}*/

	workspaceSnapshot = models.WorkspaceSnapshot{
		Workspace: entities.Workspace{
			ID:          workspaceID,
			Name:        payload.Name,
			Slug:        slug,
			IconURL:     payload.IconURL,
			Description: payload.Description,
			Timestamps: entities.Timestamps{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		SnapshotFields: models.SnapshotFields{
			SnapVersion:   1,
			SnapUpdatedAt: time.Now(),
		},
	}
	membershipSnapshot := models.WorkspaceMembershipSnapshot{
		WorkspaceMembership: entities.WorkspaceMembership{
			ID:          uuid.New(),
			WorkspaceID: workspaceID,
			UserID:      userID,
			Role:        entities.WorkspaceMemberRoleOwner,
			JoinedAt:    time.Now(),
		},
		SnapshotFields: models.SnapshotFields{
			SnapVersion:   1,
			SnapUpdatedAt: time.Now(),
		},
	}

	recordPayload := eventstorepayloads.WorkspaceCreatedPayload{
		Workspace:  workspaceSnapshot,
		Membership: membershipSnapshot,
	}

	recordPayloadBytes, err := json.Marshal(recordPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal event payload: %w", err)
	}

	record := models.EventStore{
		EventID:       messageID.String(),
		AggregateID:   workspaceID,
		AggregateType: shared.ActionKeyResourceWorkspace,
		Version:       1,
		ActionKey:     command.ActionKey,
		Payload:       recordPayloadBytes,
		Metadata:      command.Metadata,
		CreatedAt:     time.Now(),
	}

	if err := s.tx.RunInTransaction(ctx, func(ctx context.Context) error {
		// Insert event into EventStore
		if err := s.EventStoreRepo.InsertEventTX(ctx, record); err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
		// Upsert snapshot
		if err := s.SnapshotRepo.UpsertWorkspaceSnapshotTX(ctx, workspaceSnapshot); err != nil {
			return fmt.Errorf("failed to upsert workspace snapshot: %w", err)
		}
		if err := s.SnapshotRepo.UpsertMembershipSnapshotTX(ctx, membershipSnapshot); err != nil {
			return fmt.Errorf("failed to upsert membership snapshot: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	domainPayload := domaineventpayloads.WorkspaceCreatedPayload{
		WorkspaceID: workspaceID,
		UserID:      userID,
	}
	domainPayloadBytes, err := json.Marshal(domainPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal domain event payload: %w", err)
	}

	event := shared.NewEvent(
		messageID,
		workspaceID,
		command.ActionKey,
		shared.NewWorkspaceEventPartitionKey(workspaceID.String()),
		map[shared.EntityKeys][]uuid.UUID{
			shared.EntityKeysWorkspace: {workspaceID},
			shared.EntityKeysUser:      {userID},
		},
		domainPayloadBytes,
		command.GetAuthorID(),
	)

	if err := s.ProduceEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to produce event: %w", err)
	}
	s.LogChan <- fmt.Sprintf("[Service] Successfully handled CreateWorkspace command for workspace ID: %s with Name: %s", command.GetAggregateID(), payload.Name)

	return nil
}
