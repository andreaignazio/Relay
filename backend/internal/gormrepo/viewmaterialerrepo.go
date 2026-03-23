package gormrepo

import (
	"context"
	"gokafka/internal/models/materializedviews"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func (r *Repository) UpsertWorkspaceView(ctx context.Context, view materializedviews.WorkspaceView) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&view).Error
}

func (r *Repository) GetUserWorkspacesViews(ctx context.Context, userID uuid.UUID) ([]materializedviews.WorkspaceView, error) {
	var views []materializedviews.WorkspaceView
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&views).Error
	return views, err
}

func (r *Repository) UpsertChannelMembershipView(ctx context.Context, view materializedviews.ChannelMembershipView) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&view).Error
}

func (r *Repository) UpsertChannelView(ctx context.Context, view materializedviews.ChannelView) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&view).Error
}

func (r *Repository) UpsertDirectMessageMembershipView(ctx context.Context, view materializedviews.DirectMessageMembershipView) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&view).Error
}

func (r *Repository) GetUserChannelsViews(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID) ([]materializedviews.ChannelMembershipView, error) {
	var views []materializedviews.ChannelMembershipView
	err := r.db.WithContext(ctx).Where("workspace_id = ? AND user_id = ?", workspaceID, userID).Find(&views).Error
	return views, err
}

func (r *Repository) BrowseChannelsViews(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID) ([]materializedviews.ChannelView, error) {
	var views []materializedviews.ChannelView
	err := r.db.WithContext(ctx).Where("workspace_id = ? AND type != 'dm'", workspaceID).Find(&views).Error
	return views, err
}

func (r *Repository) GetUserDirectMessagesViews(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID) ([]materializedviews.DirectMessageMembershipView, error) {
	var views []materializedviews.DirectMessageMembershipView
	err := r.db.WithContext(ctx).Raw(
		`SELECT * FROM direct_message_membership_views
		WHERE channel_id IN (
			SELECT channel_id FROM direct_message_membership_views
			WHERE workspace_id = ? AND user_id = ?
		)
	`, workspaceID, userID).Scan(&views).Error
	return views, err
}
