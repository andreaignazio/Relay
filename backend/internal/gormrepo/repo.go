package gormrepo

import (
	"context"
	"gokafka/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

type txKey struct{}

func (r *Repository) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func (r *Repository) dbFromCtx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return r.db
}

func (r *Repository) InsertEventTX(ctx context.Context, event models.EventStore) error {
	return r.dbFromCtx(ctx).Create(&event).Error
}

func (r *Repository) GetStoredEventByID(ctx context.Context, eventID uuid.UUID) (models.EventStore, error) {
	var event models.EventStore
	err := r.db.WithContext(ctx).First(&event, "event_id = ?", eventID).Error
	return event, err
}

func (r *Repository) GetSnapshot(ctx context.Context, aggregateID uuid.UUID, snapshot *models.WorkspaceSnapshot) error {
	return r.db.WithContext(ctx).Where("aggregate_id = ?", aggregateID).First(snapshot).Error
}

func (r *Repository) UpsertWorkspaceSnapshotTX(ctx context.Context, snapshot models.WorkspaceSnapshot) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&snapshot).Error
}

func (r *Repository) UpsertMembershipSnapshotTX(ctx context.Context, snapshot models.WorkspaceMembershipSnapshot) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&snapshot).Error
}

func (r *Repository) GetChannelSnapshot(ctx context.Context, aggregateID uuid.UUID, snapshot *models.ChannelSnapshot) error {
	return r.db.WithContext(ctx).Where("id = ?", aggregateID).First(snapshot).Error
}

func (r *Repository) UpsertChannelSnapshotTX(ctx context.Context, snapshot models.ChannelSnapshot) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&snapshot).Error
}

func (r *Repository) UpsertChannelMembershipSnapshotTX(ctx context.Context, snapshot models.ChannelMembershipSnapshot) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&snapshot).Error
}

func (r *Repository) CheckWorkspaceSlugExists(ctx context.Context, slug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.WorkspaceSnapshot{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
