package entities

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// UUIDArray is a []uuid.UUID that serializes to/from a PostgreSQL uuid[] column.
// It returns "{}" for nil/empty slices instead of NULL, satisfying NOT NULL constraints.
type UUIDArray []uuid.UUID

func (a UUIDArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	parts := make([]string, len(a))
	for i, id := range a {
		parts[i] = id.String()
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

func (a *UUIDArray) Scan(src interface{}) error {
	if src == nil {
		*a = UUIDArray{}
		return nil
	}
	var str string
	switch v := src.(type) {
	case string:
		str = v
	case []byte:
		str = string(v)
	default:
		return fmt.Errorf("UUIDArray.Scan: unsupported type %T", src)
	}
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*a = UUIDArray{}
		return nil
	}
	parts := strings.Split(str, ",")
	ids := make(UUIDArray, len(parts))
	for i, p := range parts {
		id, err := uuid.Parse(strings.TrimSpace(p))
		if err != nil {
			return fmt.Errorf("UUIDArray.Scan: failed to parse UUID %q: %w", p, err)
		}
		ids[i] = id
	}
	*a = ids
	return nil
}

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
