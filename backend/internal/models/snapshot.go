package models

import (
	"gokafka/internal/models/entities"
	"time"
)

type SnapshotFields struct {
	SnapVersion   int
	SnapUpdatedAt time.Time
}

type UserSnapshot struct {
	entities.User
	SnapshotFields
}

type WorkspaceSnapshot struct {
	entities.Workspace
	SnapshotFields
}

type WorkspaceMembershipSnapshot struct {
	entities.WorkspaceMembership
	SnapshotFields
}

type ChannelSnapshot struct {
	entities.Channel
	SnapshotFields
}

type ChannelMembershipSnapshot struct {
	entities.ChannelMembership
	SnapshotFields
}

type MessageSnapshot struct {
	entities.Message
	SnapshotFields
}

type ReactionSnapshot struct {
	entities.Reaction
	SnapshotFields
}

type AttachmentSnapshot struct {
	entities.Attachment
	SnapshotFields
}

type NotificationSnapshot struct {
	entities.Notification
	SnapshotFields
}

type InviteSnapshot struct {
	entities.Invite
	SnapshotFields
}
