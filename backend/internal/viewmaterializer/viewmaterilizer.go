package viewmaterializer

import (
	"context"
	"encoding/json"
	"fmt"
	"gokafka/internal/eventstorepayloads"
	"gokafka/internal/models"
	"gokafka/internal/models/entities"
	"gokafka/internal/models/materializedviews"
	"gokafka/internal/shared"

	"github.com/google/uuid"
)

type Service struct {
	EventStoreRepo          EventStoreRepo
	WorkspaceViewRepository WorkspaceViewRepository
	ChannelViewRepository   ChannelViewRepository
	tx                      Transactor
}

func NewService(eventStoreRepo EventStoreRepo, workspaceViewRepo WorkspaceViewRepository, channelViewRepo ChannelViewRepository, tx Transactor) *Service {
	return &Service{
		EventStoreRepo:          eventStoreRepo,
		WorkspaceViewRepository: workspaceViewRepo,
		ChannelViewRepository:   channelViewRepo,
		tx:                      tx,
	}
}

type Transactor interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type EventStoreRepo interface {
	GetStoredEventByID(ctx context.Context, eventID uuid.UUID) (models.EventStore, error)
}

type WorkspaceViewRepository interface {
	UpsertWorkspaceView(ctx context.Context, view materializedviews.WorkspaceView) error
}

type ChannelViewRepository interface {
	UpsertChannelMembershipView(ctx context.Context, view materializedviews.ChannelMembershipView) error
	UpsertChannelView(ctx context.Context, view materializedviews.ChannelView) error
	UpsertDirectMessageMembershipView(ctx context.Context, view materializedviews.DirectMessageMembershipView) error
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
			Description: payload.Workspace.Description,
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

func (s *Service) HandleChannelViewUpdate(ctx context.Context, event shared.Event) error {

	switch event.ActionKey {
	case shared.ActionKeyChannelCreate:
		messageID := event.GetMessageID()
		storedEvent, err := s.EventStoreRepo.GetStoredEventByID(ctx, messageID)
		if err != nil {
			return fmt.Errorf("failed to get event by ID: %w", err)
		}
		var storedPayload eventstorepayloads.ChannelCreatedPayload
		if err := json.Unmarshal(storedEvent.Payload, &storedPayload); err != nil {
			return fmt.Errorf("failed to unmarshal event payload: %w", err)
		}

		switch storedPayload.Channel.Type {
		case entities.ChannelTypePublic, entities.ChannelTypePrivate:

			channelMembersView := materializedviews.ChannelMembershipView{
				ChannelID:         storedPayload.Channel.ID,
				UserID:            storedPayload.Membership.UserID,
				WorkspaceID:       storedPayload.Channel.WorkspaceID,
				Name:              storedPayload.Channel.Name,
				Description:       storedPayload.Channel.Description,
				Topic:             storedPayload.Channel.Topic,
				Type:              storedPayload.Channel.Type,
				IsArchived:        storedPayload.Channel.IsArchived,
				CreatedBy:         storedPayload.Channel.CreatedBy,
				Role:              storedPayload.Membership.Role,
				JoinedAt:          storedPayload.Membership.JoinedAt,
				NotificationsPref: storedPayload.Membership.NotificationsPref,
				LastReadAt:        storedPayload.Membership.LastReadAt,
			}

			channelView := materializedviews.ChannelView{
				ChannelID:   storedPayload.Channel.ID,
				WorkspaceID: storedPayload.Channel.WorkspaceID,
				CreatedBy:   storedPayload.Channel.CreatedBy,
				Name:        storedPayload.Channel.Name,
				Description: storedPayload.Channel.Description,
				Topic:       storedPayload.Channel.Topic,
				Type:        storedPayload.Channel.Type,
				IsArchived:  storedPayload.Channel.IsArchived,
			}

			// Handle standard channel creation event to update the channel view
			err = s.tx.RunInTransaction(ctx, func(ctx context.Context) error {
				if err := s.ChannelViewRepository.UpsertChannelMembershipView(ctx, channelMembersView); err != nil {
					return fmt.Errorf("failed to upsert channel membership view: %w", err)
				}
				if err := s.ChannelViewRepository.UpsertChannelView(ctx, channelView); err != nil {
					return fmt.Errorf("failed to upsert channel view: %w", err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("failed to upsert channel views in transaction: %w", err)
			}
			return nil

		}
	case shared.ActionKeyChannelUpdate:
		fmt.Println("Channel membership update event received. Event handling logic to be implemented.")
		return nil
	}
	return nil
}

func (s *Service) HandleDMViewUpdate(ctx context.Context, event shared.Event) error {
	switch event.ActionKey {
	case shared.ActionKeyDMCreate:
		messageID := event.GetMessageID()
		storedEvent, err := s.EventStoreRepo.GetStoredEventByID(ctx, messageID)
		if err != nil {
			return fmt.Errorf("failed to get event by ID: %w", err)
		}
		var payload eventstorepayloads.DMCreatedPayload
		if err := json.Unmarshal(storedEvent.Payload, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal event payload: %w", err)
		}
		for _, ms := range payload.Memberships {
			view := materializedviews.DirectMessageMembershipView{
				ChannelID:         payload.Channel.ID,
				UserID:            ms.UserID,
				WorkspaceID:       payload.Channel.WorkspaceID,
				JoinedAt:          ms.JoinedAt,
				NotificationsPref: ms.NotificationsPref,
			}
			if err := s.ChannelViewRepository.UpsertDirectMessageMembershipView(ctx, view); err != nil {
				return fmt.Errorf("failed to upsert DM membership view: %w", err)
			}
		}
	}
	return nil
}
