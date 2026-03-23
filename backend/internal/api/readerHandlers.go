package api

import (
	"context"
	"gokafka/internal/models/materializedviews"

	"github.com/gin-gonic/gin"
)

type ReaderApiHandler struct {
	MaterialedViewRepository MaterialedViewRepository
}

type MaterialedViewRepository interface {
	GetUserWorkspacesViews(ctx context.Context, userID string) ([]materializedviews.WorkspaceView, error)
}

func NewReaderHandler(MaterialedViewRepository MaterialedViewRepository) *ReaderApiHandler {
	return &ReaderApiHandler{
		MaterialedViewRepository: MaterialedViewRepository,
	}
}

func (h *ReaderApiHandler) ListUserWorkspaces(c *gin.Context) {
	userID := c.MustGet("UserID").(string)
	ctx := c.Request.Context()
	views, err := h.MaterialedViewRepository.GetUserWorkspacesViews(ctx, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, views)
}
