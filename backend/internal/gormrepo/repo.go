package gormrepo

import (
	"context"
	"gokafka/internal/models"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) CreateWorkspace(ctx context.Context, workspace *models.Workspace) error {
	return r.db.WithContext(ctx).Create(workspace).Error
}
