package api

import (
	"gokafka/internal/models"
	"gokafka/internal/models/materializedviews"
	"time"

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

// --- Members ---

type MemberInfo struct {
	UserID   uuid.UUID `json:"UserId"`
	Role     string    `json:"Role"`
	JoinedAt time.Time `json:"JoinedAt"`
}

type MemberProfile struct {
	UserID      uuid.UUID `json:"UserId"`
	Username    string    `json:"Username"`
	DisplayName string    `json:"DisplayName"`
	AvatarURL   *string   `json:"AvatarUrl,omitempty"`
	Role        string    `json:"Role"`
	JoinedAt    time.Time `json:"JoinedAt"`
}

type UserProfile struct {
	UserID      uuid.UUID `json:"UserId"`
	Username    string    `json:"Username"`
	DisplayName string    `json:"DisplayName"`
	AvatarURL   *string   `json:"AvatarUrl,omitempty"`
}

type ListWorkspaceMembersResponse struct {
	Members []MemberProfile `json:"Members"`
}

type ListMemberIDsResponse struct {
	Members []MemberInfo `json:"Members"`
}

type BatchUsersResponse struct {
	Users []UserProfile `json:"Users"`
}
