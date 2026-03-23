package api

import (
	"context"
	"gokafka/internal/models/materializedviews"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReaderApiHandler struct {
	MaterialedViewRepository MaterialedViewRepository
}

type MaterialedViewRepository interface {
	GetUserWorkspacesViews(ctx context.Context, userID uuid.UUID) ([]materializedviews.WorkspaceView, error)
	GetUserChannelsViews(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID) ([]materializedviews.ChannelMembershipView, error)
	GetUserDirectMessagesViews(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID) ([]materializedviews.DirectMessageMembershipView, error)
	BrowseChannelsViews(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID) ([]materializedviews.ChannelView, error)
}

func NewReaderHandler(MaterialedViewRepository MaterialedViewRepository) *ReaderApiHandler {
	return &ReaderApiHandler{
		MaterialedViewRepository: MaterialedViewRepository,
	}
}

func (h *ReaderApiHandler) ListUserWorkspaces(c *gin.Context) {
	userID, err := uuid.Parse(c.MustGet("UserID").(string))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}
	ctx := c.Request.Context()
	views, err := h.MaterialedViewRepository.GetUserWorkspacesViews(ctx, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, ListUserWorkspacesResponse{Workspaces: views})
}

func (h *ReaderApiHandler) ListUserChannels(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("workspaceID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid workspace ID"})
		return
	}
	userID, err := uuid.Parse(c.MustGet("UserID").(string))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}
	ctx := c.Request.Context()
	views, err := h.MaterialedViewRepository.GetUserChannelsViews(ctx, workspaceID, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, ListUserChannelsResponse{Channels: views})
}

func (h *ReaderApiHandler) ListUserDirectMessages(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("workspaceID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid workspace ID"})
		return
	}
	userID, err := uuid.Parse(c.MustGet("UserID").(string))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}
	ctx := c.Request.Context()
	views, err := h.MaterialedViewRepository.GetUserDirectMessagesViews(ctx, workspaceID, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	viewsByChannelID := make(map[uuid.UUID][]materializedviews.DirectMessageMembershipView)
	for _, view := range views {
		viewsByChannelID[view.ChannelID] = append(viewsByChannelID[view.ChannelID], view)
	}
	c.JSON(200, ListUserDirectMessagesResponse{DirectMessages: viewsByChannelID})
}

func (h *ReaderApiHandler) BrowseChannels(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("workspaceID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid workspace ID"})
		return
	}
	userID, err := uuid.Parse(c.MustGet("UserID").(string))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}
	ctx := c.Request.Context()
	views, err := h.MaterialedViewRepository.BrowseChannelsViews(ctx, workspaceID, userID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, BrowseChannelsResponse{Channels: views})
}
