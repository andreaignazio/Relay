package workspaces

import (
	"context"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/models"
	"gokafka/internal/shared"

	"github.com/google/uuid"
)

type Service struct {
	tx             Transactor
	EventStoreRepo EventStoreRepo
	SnapshotRepo   SnapshotRepo
	Producer       *producer.KafkaProducer
	LogChan        chan string
}

type Transactor interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type EventStoreRepo interface {
	InsertEventTX(ctx context.Context, event models.EventStore) error
}

type SnapshotRepo interface {
	GetSnapshot(ctx context.Context, aggregateID uuid.UUID, snapshot *models.WorkspaceSnapshot) error
	CheckWorkspaceSlugExists(ctx context.Context, slug string) (bool, error)
	UpsertWorkspaceSnapshotTX(ctx context.Context, snapshot models.WorkspaceSnapshot) error
	UpsertMembershipSnapshotTX(ctx context.Context, snapshot models.WorkspaceMembershipSnapshot) error
}

func NewService(tx Transactor, eventStoreRepo EventStoreRepo, snapshotRepo SnapshotRepo, producer *producer.KafkaProducer, logChan chan string) *Service {
	return &Service{
		tx:             tx,
		EventStoreRepo: eventStoreRepo,
		SnapshotRepo:   snapshotRepo,
		Producer:       producer,
		LogChan:        logChan,
	}
}

func (s *Service) ProduceEvent(ctx context.Context, event shared.Event) error {
	if err := s.Producer.WriteMessage(ctx, event); err != nil {
		return err
	}
	return nil

}
