package entities

type UserAuthProvider string

const (
	AuthProviderLocal  UserAuthProvider = "local"
	AuthProviderGoogle UserAuthProvider = "google"
	AuthProviderGithub UserAuthProvider = "github"
)

type WorkspaceMemberRole string

const (
	WorkspaceMemberRoleOwner  WorkspaceMemberRole = "owner"
	WorkspaceMemberRoleAdmin  WorkspaceMemberRole = "admin"
	WorkspaceMemberRoleMember WorkspaceMemberRole = "member"
	WorkspaceMemberRoleGuest  WorkspaceMemberRole = "guest"
)

type ChannelType string

const (
	ChannelTypePublic  ChannelType = "public"
	ChannelTypePrivate ChannelType = "private"
	ChannelTypeDM      ChannelType = "dm"
)

type ChannelMemberRole string

const (
	ChannelMemberRoleAdmin  ChannelMemberRole = "admin"
	ChannelMemberRoleMember ChannelMemberRole = "member"
)

type NotificationPref string

const (
	NotificationPrefAll      NotificationPref = "all"
	NotificationPrefMentions NotificationPref = "mentions"
	NotificationPrefNone     NotificationPref = "none"
)

type NotificationType string

const (
	NotificationTypeMention       NotificationType = "mention"
	NotificationTypeThreadReply   NotificationType = "thread_reply"
	NotificationTypeChannelInvite NotificationType = "channel_invite"
)
