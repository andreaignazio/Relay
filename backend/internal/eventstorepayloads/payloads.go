package eventstorepayloads

import (
	"gokafka/internal/models"
)

type WorkspaceCreatedPayload struct {
	Workspace  models.WorkspaceSnapshot
	Membership models.WorkspaceMembershipSnapshot
}

type ChannelCreatedPayload struct {
	Channel    models.ChannelSnapshot
	Membership models.ChannelMembershipSnapshot
}

type MessageCreatedPayload struct {
	Message models.MessageSnapshot
}

type DMCreatedPayload struct {
	Channel     models.ChannelSnapshot
	Memberships []models.ChannelMembershipSnapshot
}

type UserRegisteredPayload struct {
	User models.UserSnapshot
}
