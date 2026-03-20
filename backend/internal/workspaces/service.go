package workspaces

import (
	"context"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/models"
	"gokafka/internal/shared"
)

type Service struct {
	Repo     Repo
	Producer *producer.KafkaProducer
	LogChan  chan string
}

type Repo interface {
	CreateWorkspace(ctx context.Context, workspace *models.Workspace) error
	// Define repository methods here
}

func NewService(repo Repo, producer *producer.KafkaProducer, logChan chan string) *Service {
	return &Service{
		Repo:     repo,
		Producer: producer,
		LogChan:  logChan,
	}
}

func (s *Service) ProduceEvent(ctx context.Context, event shared.Event) error {
	if err := s.Producer.WriteMessage(ctx, event); err != nil {
		return err
	}
	return nil

}
