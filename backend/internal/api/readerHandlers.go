package api

import (
	"context"
	"gokafka/internal/models"
	"gokafka/internal/models/materializedviews"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReaderApiHandler struct {
	MaterialedViewRepository MaterialedViewRepository
	SnapshotRepository       SnapshotRepository
}

type MaterialedViewRepository interface {
	GetUserWorkspacesViews(ctx context.Context, userID uuid.UUID) ([]materializedviews.WorkspaceView, error)
	GetUserChannelsViews(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID) ([]materializedviews.ChannelMembershipView, error)
	GetUserDirectMessagesViews(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID) ([]materializedviews.DirectMessageMembershipView, error)
	BrowseChannelsViews(ctx context.Context, workspaceID uuid.UUID, userID uuid.UUID) ([]materializedviews.ChannelView, error)
	GetWorkspaceMemberViews(ctx context.Context, workspaceID uuid.UUID) ([]materializedviews.WorkspaceView, error)
	GetChannelMemberViews(ctx context.Context, channelID uuid.UUID) ([]materializedviews.ChannelMembershipView, error)
}

type SnapshotRepository interface {
	GetChannelMessagesSnapshot(ctx context.Context, channelID uuid.UUID) ([]models.MessageSnapshot, error)
	GetUserSnapshotsByIDs(ctx context.Context, userIDs []uuid.UUID) ([]models.UserSnapshot, error)
}

func NewReaderHandler(MaterialedViewRepository MaterialedViewRepository, SnapshotRepository SnapshotRepository) *ReaderApiHandler {
	return &ReaderApiHandler{
		MaterialedViewRepository: MaterialedViewRepository,
		SnapshotRepository:       SnapshotRepository,
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

func (h *ReaderApiHandler) ListChannelMessages(c *gin.Context) {
	_, err := uuid.Parse(c.Param("workspaceID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid workspace ID"})
		return
	}
	channelID, err := uuid.Parse(c.Param("channelID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid channel ID"})
		return
	}
	_, err = uuid.Parse(c.MustGet("UserID").(string))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}
	ctx := c.Request.Context()
	messages, err := h.SnapshotRepository.GetChannelMessagesSnapshot(ctx, channelID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, ListChannelMessagesResponse{Messages: messages})
}

func (h *ReaderApiHandler) ListWorkspaceMembers(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("workspaceID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid workspace ID"})
		return
	}
	_, err = uuid.Parse(c.MustGet("UserID").(string))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}
	ctx := c.Request.Context()

	views, err := h.MaterialedViewRepository.GetWorkspaceMemberViews(ctx, workspaceID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	userIDs := make([]uuid.UUID, len(views))
	for i, v := range views {
		userIDs[i] = v.UserID
	}

	users, err := h.SnapshotRepository.GetUserSnapshotsByIDs(ctx, userIDs)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	userMap := make(map[uuid.UUID]models.UserSnapshot, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	members := make([]MemberProfile, 0, len(views))
	for _, v := range views {
		u, ok := userMap[v.UserID]
		if !ok {
			continue
		}
		members = append(members, MemberProfile{
			UserID:      v.UserID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			AvatarURL:   u.AvatarURL,
			Role:        string(v.Role),
			JoinedAt:    v.JoinedAt,
		})
	}

	c.JSON(200, ListWorkspaceMembersResponse{Members: members})
}

func (h *ReaderApiHandler) ListWorkspaceMemberIDs(c *gin.Context) {
	workspaceID, err := uuid.Parse(c.Param("workspaceID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid workspace ID"})
		return
	}
	_, err = uuid.Parse(c.MustGet("UserID").(string))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}
	ctx := c.Request.Context()

	views, err := h.MaterialedViewRepository.GetWorkspaceMemberViews(ctx, workspaceID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	members := make([]MemberInfo, len(views))
	for i, v := range views {
		members[i] = MemberInfo{
			UserID:   v.UserID,
			Role:     string(v.Role),
			JoinedAt: v.JoinedAt,
		}
	}

	c.JSON(200, ListMemberIDsResponse{Members: members})
}

func (h *ReaderApiHandler) ListChannelMemberIDs(c *gin.Context) {
	_, err := uuid.Parse(c.Param("workspaceID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid workspace ID"})
		return
	}
	channelID, err := uuid.Parse(c.Param("channelID"))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid channel ID"})
		return
	}
	_, err = uuid.Parse(c.MustGet("UserID").(string))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}
	ctx := c.Request.Context()

	views, err := h.MaterialedViewRepository.GetChannelMemberViews(ctx, channelID)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	members := make([]MemberInfo, len(views))
	for i, v := range views {
		members[i] = MemberInfo{
			UserID:   v.UserID,
			Role:     string(v.Role),
			JoinedAt: v.JoinedAt,
		}
	}

	c.JSON(200, ListMemberIDsResponse{Members: members})
}

func (h *ReaderApiHandler) BatchGetUsers(c *gin.Context) {
	_, err := uuid.Parse(c.MustGet("UserID").(string))
	if err != nil {
		c.JSON(400, gin.H{"error": "invalid user ID"})
		return
	}

	var req BatchUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	userIDs := make([]uuid.UUID, len(req.UserIds))
	for i, id := range req.UserIds {
		parsed, err := uuid.Parse(id)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid user ID: " + id})
			return
		}
		userIDs[i] = parsed
	}

	ctx := c.Request.Context()
	snapshots, err := h.SnapshotRepository.GetUserSnapshotsByIDs(ctx, userIDs)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	users := make([]UserProfile, len(snapshots))
	for i, s := range snapshots {
		users[i] = UserProfile{
			UserID:      s.ID,
			Username:    s.Username,
			DisplayName: s.DisplayName,
			AvatarURL:   s.AvatarURL,
		}
	}

	c.JSON(200, BatchUsersResponse{Users: users})
}
