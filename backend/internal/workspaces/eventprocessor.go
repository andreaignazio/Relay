package workspaces

import (
	"context"
	"fmt"
	"gokafka/internal/models"
	"gokafka/internal/shared"
)

func (s *Service) HandleCreateWorkspace(ctx context.Context, event shared.Command) error {
	//fmt.Println("[Service] Handling CreateWorkspace command for workspace ID:", event.GetPartitionKey())
	s.LogChan <- fmt.Sprintf("[Service] Handling CreateWorkspace command for workspace ID: %s", event.GetPartitionKey())
	workspaceID := event.GetPartitionKey().ID()

	if err := s.Repo.CreateWorkspace(ctx, &models.Workspace{
		ID:   workspaceID,
		Name: "Test Workspace",
	}); err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	newEvent := shared.NewEvent(
		shared.ActionKeyWorkspaceCreate,
		shared.NewWorkspaceEventPartitionKey(workspaceID),
		map[string]string{"name": "Test Workspace"},
	)
	if err := s.ProduceEvent(ctx, newEvent); err != nil {
		return fmt.Errorf("failed to produce event: %w", err)
	}
	return nil
}
