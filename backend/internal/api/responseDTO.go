package api

import (
	"gokafka/internal/models"
	"gokafka/internal/models/materializedviews"

	"github.com/google/uuid"
)

type ListUserWorkspacesResponse struct {
	Workspaces []materializedviews.WorkspaceView `json:"Workspaces"`
}

type ListUserChannelsResponse struct {
	Channels []materializedviews.ChannelMembershipView `json:"Channels"`
}

type ListUserDirectMessagesResponse struct {
	DirectMessages map[uuid.UUID][]materializedviews.DirectMessageMembershipView `json:"DirectMessages"`
}

type BrowseChannelsResponse struct {
	Channels []materializedviews.ChannelView `json:"Channels"`
}

type ListChannelMessagesResponse struct {
	Messages []models.MessageSnapshot `json:"Messages"`
}
