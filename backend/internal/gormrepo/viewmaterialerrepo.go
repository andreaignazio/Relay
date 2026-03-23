package gormrepo

import (
	"context"
	"gokafka/internal/models/materializedviews"

	"gorm.io/gorm/clause"
)

func (r *Repository) UpsertWorkspaceView(ctx context.Context, view materializedviews.WorkspaceView) error {
	return r.dbFromCtx(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Create(&view).Error
}

func (r *Repository) GetUserWorkspacesViews(ctx context.Context, userID string) ([]materializedviews.WorkspaceView, error) {
	var views []materializedviews.WorkspaceView
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&views).Error
	return views, err
}
