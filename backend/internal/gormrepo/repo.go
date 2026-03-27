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

func (r *Repository) GetMessageSnapshot(ctx context.Context, aggregateID uuid.UUID, snapshot *models.MessageSnapshot) error {
	return r.db.WithContext(ctx).Where("id = ?", aggregateID).First(snapshot).Error
}

func (r *Repository) GetChannelMessagesSnapshot(ctx context.Context, channelID uuid.UUID) ([]models.MessageSnapshot, error) {
	var snapshots []models.MessageSnapshot
	err := r.db.WithContext(ctx).Where("channel_id = ?", channelID).Order("created_at ASC").Find(&snapshots).Error
	return snapshots, err
}

func (r *Repository) GetUserSnapshot(ctx context.Context, userID uuid.UUID, snapshot *models.UserSnapshot) error {
	return r.db.WithContext(ctx).Where("id = ?", userID).First(snapshot).Error
}

func (r *Repository) UpsertMessageSnapshotTX(ctx context.Context, snapshot models.MessageSnapshot) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&snapshot).Error
}

func (r *Repository) GetUserSnapshotsByIDs(ctx context.Context, userIDs []uuid.UUID) ([]models.UserSnapshot, error) {
	var snapshots []models.UserSnapshot
	err := r.db.WithContext(ctx).Where("id IN (?)", userIDs).Find(&snapshots).Error
	return snapshots, err
}

func (r *Repository) FindExistingDMChannel(ctx context.Context, workspaceID uuid.UUID, participantIDs []uuid.UUID) (uuid.UUID, bool, error) {
	var channelID uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT cs.id
		FROM channel_snapshots cs
		JOIN channel_membership_snapshots cms ON cs.id = cms.channel_id
		WHERE cs.workspace_id = ?
		  AND cs.type = 'dm'
		  AND cs.deleted_at IS NULL
		  AND cms.deleted_at IS NULL
		  AND cms.user_id IN (?)
		GROUP BY cs.id
		HAVING COUNT(DISTINCT cms.user_id) = ?
		AND cs.id NOT IN (
			SELECT cms2.channel_id
			FROM channel_membership_snapshots cms2
			JOIN channel_snapshots cs2 ON cs2.id = cms2.channel_id
			WHERE cs2.workspace_id = ?
			  AND cs2.type = 'dm'
			  AND cs2.deleted_at IS NULL
			  AND cms2.deleted_at IS NULL
			  AND cms2.user_id NOT IN (?)
		)
		LIMIT 1
	`, workspaceID, participantIDs, len(participantIDs), workspaceID, participantIDs).Scan(&channelID).Error
	if err != nil {
		return uuid.Nil, false, err
	}
	if channelID == uuid.Nil {
		return uuid.Nil, false, nil
	}
	return channelID, true, nil
}

func (r *Repository) UpsertUserSnapshotTX(ctx context.Context, snapshot models.UserSnapshot) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&snapshot).Error
}

func (r *Repository) CheckUsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.UserSnapshot{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) CheckWorkspaceSlugExists(ctx context.Context, slug string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.WorkspaceSnapshot{}).Where("slug = ?", slug).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
