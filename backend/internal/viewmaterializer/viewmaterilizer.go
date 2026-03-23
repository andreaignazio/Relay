package viewmaterializer

import (
	"context"
	"encoding/json"
	"fmt"
	"gokafka/internal/eventstorepayloads"
	"gokafka/internal/models"
	"gokafka/internal/models/materializedviews"
	"gokafka/internal/shared"

	"github.com/google/uuid"
)

type Service struct {
	EventStoreRepo          EventStoreRepo
	WorkspaceViewRepository WorkspaceViewRepository
}

func NewService(eventStoreRepo EventStoreRepo, workspaceViewRepo WorkspaceViewRepository) *Service {
	return &Service{
		EventStoreRepo:          eventStoreRepo,
		WorkspaceViewRepository: workspaceViewRepo,
	}
}

type EventStoreRepo interface {
	GetStoredEventByID(ctx context.Context, eventID uuid.UUID) (models.EventStore, error)
}

type WorkspaceViewRepository interface {
	UpsertWorkspaceView(ctx context.Context, view materializedviews.WorkspaceView) error
}

func (s *Service) HandleWorkspaceViewUpdate(ctx context.Context, event shared.Event) error {

	switch event.ActionKey {
	case shared.ActionKeyWorkspaceCreate:

		messageID := event.GetMessageID()
		storedEvent, err := s.EventStoreRepo.GetStoredEventByID(ctx, messageID)
		if err != nil {
			return fmt.Errorf("failed to get event by ID: %w", err)
		}
		var payload eventstorepayloads.WorkspaceCreatedPayload
		if err := json.Unmarshal(storedEvent.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal event payload: %w", err)
		}

		workspaceViewUpdate := materializedviews.WorkspaceView{
			WorkspaceID: payload.Workspace.ID,
			UserID:      payload.Membership.UserID,
			Name:        payload.Workspace.Name,
			Slug:        payload.Workspace.Slug,
			IconURL:     payload.Workspace.IconURL,
			CreatedAt:   payload.Workspace.CreatedAt,
			UpdatedAt:   payload.Workspace.UpdatedAt,
			Role:        payload.Membership.Role,
			JoinedAt:    payload.Membership.JoinedAt,
		}
		//fmt.Print("Workspace view update error: ", workspaceViewUpdate)
		if err := s.WorkspaceViewRepository.UpsertWorkspaceView(ctx, workspaceViewUpdate); err != nil {
			return fmt.Errorf("failed to upsert workspace view: %w", err)
		}

		// Handle workspace creation event to update the view
		// This would involve reading the event payload, extracting necessary information,
		// and then updating the read model (e.g., a denormalized table) accordingly.
		// For example, you might insert a new record into a "workspaces_view" table with the workspace details.
	}

	return nil
}
