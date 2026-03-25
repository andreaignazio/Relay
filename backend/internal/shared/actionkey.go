package shared

import "fmt"

// --- Types ---

type ActionKey string

type ActionKeyVerb string

const (
	ActionKeyVerbAccept     ActionKeyVerb = "accept"
	ActionKeyVerbAdd        ActionKeyVerb = "add"
	ActionKeyVerbArchive    ActionKeyVerb = "archive"
	ActionKeyVerbClear      ActionKeyVerb = "clear"
	ActionKeyVerbCreate     ActionKeyVerb = "create"
	ActionKeyVerbDeactivate ActionKeyVerb = "deactivate"
	ActionKeyVerbDelete     ActionKeyVerb = "delete"
	ActionKeyVerbEdit       ActionKeyVerb = "edit"
	ActionKeyVerbGet        ActionKeyVerb = "get"
	ActionKeyVerbJoin       ActionKeyVerb = "join"
	ActionKeyVerbKick       ActionKeyVerb = "kick"
	ActionKeyVerbLeave      ActionKeyVerb = "leave"
	ActionKeyVerbList       ActionKeyVerb = "list"
	ActionKeyVerbLogin      ActionKeyVerb = "login"
	ActionKeyVerbMarkRead   ActionKeyVerb = "mark_read"
	ActionKeyVerbPin        ActionKeyVerb = "pin"
	ActionKeyVerbRead       ActionKeyVerb = "read"
	ActionKeyVerbReadAll    ActionKeyVerb = "read_all"
	ActionKeyVerbRegister   ActionKeyVerb = "register"
	ActionKeyVerbReject     ActionKeyVerb = "reject"
	ActionKeyVerbRemove     ActionKeyVerb = "remove"
	ActionKeyVerbRevoke     ActionKeyVerb = "revoke"
	ActionKeyVerbSearch     ActionKeyVerb = "search"
	ActionKeyVerbReply      ActionKeyVerb = "reply"
	ActionKeyVerbSend       ActionKeyVerb = "send"
	ActionKeyVerbSet        ActionKeyVerb = "set"
	ActionKeyVerbUnarchive  ActionKeyVerb = "unarchive"
	ActionKeyVerbUnpin      ActionKeyVerb = "unpin"
	ActionKeyVerbUpdate     ActionKeyVerb = "update"
	ActionKeyVerbUpdateRole ActionKeyVerb = "update_role"
	ActionKeyVerbUpload     ActionKeyVerb = "upload"
)

type ActionKeyResource string

const (
	ActionKeyResourceAttachment    ActionKeyResource = "attachment"
	ActionKeyResourceChannel       ActionKeyResource = "channel"
	ActionKeyResourceChannelMember ActionKeyResource = "channel_member"
	ActionKeyResourceDM            ActionKeyResource = "dm"
	ActionKeyResourceInvite        ActionKeyResource = "invite"
	ActionKeyResourceMember        ActionKeyResource = "member"
	ActionKeyResourceMessage       ActionKeyResource = "message"
	ActionKeyResourceNotification  ActionKeyResource = "notification"
	ActionKeyResourcePlatform      ActionKeyResource = "platform"
	ActionKeyResourceReaction      ActionKeyResource = "reaction"
	ActionKeyResourceStatus        ActionKeyResource = "status"
	ActionKeyResourceUser          ActionKeyResource = "user"
	ActionKeyResourceWorkspace     ActionKeyResource = "workspace"
)

func (kr ActionKeyResource) MapParent() ActionKeyResource {
	switch kr {
	case ActionKeyResourceUser:
		return ActionKeyResourcePlatform
	case ActionKeyResourceWorkspace:
		return ActionKeyResourcePlatform
	case ActionKeyResourceMember:
		return ActionKeyResourceWorkspace
	case ActionKeyResourceChannel:
		return ActionKeyResourceWorkspace
	case ActionKeyResourceDM:
		return ActionKeyResourceWorkspace
	case ActionKeyResourceChannelMember:
		return ActionKeyResourceChannel
	case ActionKeyResourceMessage:
		return ActionKeyResourceChannel
	case ActionKeyResourceReaction:
		return ActionKeyResourceMessage
	case ActionKeyResourceAttachment:
		return ActionKeyResourceMessage
	case ActionKeyResourceNotification:
		return ActionKeyResourceUser
	case ActionKeyResourceInvite:
		return ActionKeyResourceWorkspace
	case ActionKeyResourceStatus:
		return ActionKeyResourceWorkspace
	default:
		return ""
	}
}

func NewActionKey(verb ActionKeyVerb, resource ActionKeyResource) (ActionKey, error) {
	parent := resource.MapParent()
	if parent == "" {
		return "", fmt.Errorf("invalid action key for verb %s and resource %s", verb, resource)
	}
	return ActionKey(string(resource) + ":" + string(parent) + ":" + string(verb)), nil
}

// --- Registry ---

var registeredActionKeys []ActionKey

func mustActionKey(v ActionKeyVerb, r ActionKeyResource) ActionKey {
	k, err := NewActionKey(v, r)
	if err != nil {
		panic(fmt.Sprintf("invalid action key: %v", err))
	}
	registeredActionKeys = append(registeredActionKeys, k)
	return k
}

func RegisteredActionKeys() []ActionKey {
	return registeredActionKeys
}

// --- Action Keys ---

var (
	// user
	ActionKeyUserRegister   = mustActionKey(ActionKeyVerbRegister, ActionKeyResourceUser)
	ActionKeyUserLogin      = mustActionKey(ActionKeyVerbLogin, ActionKeyResourceUser)
	ActionKeyUserGet        = mustActionKey(ActionKeyVerbGet, ActionKeyResourceUser)
	ActionKeyUserUpdate     = mustActionKey(ActionKeyVerbUpdate, ActionKeyResourceUser)
	ActionKeyUserDeactivate = mustActionKey(ActionKeyVerbDeactivate, ActionKeyResourceUser)

	// workspace
	ActionKeyWorkspaceCreate = mustActionKey(ActionKeyVerbCreate, ActionKeyResourceWorkspace)
	ActionKeyWorkspaceList   = mustActionKey(ActionKeyVerbList, ActionKeyResourceWorkspace)
	ActionKeyWorkspaceGet    = mustActionKey(ActionKeyVerbGet, ActionKeyResourceWorkspace)
	ActionKeyWorkspaceUpdate = mustActionKey(ActionKeyVerbUpdate, ActionKeyResourceWorkspace)
	ActionKeyWorkspaceDelete = mustActionKey(ActionKeyVerbDelete, ActionKeyResourceWorkspace)

	// member
	ActionKeyMemberJoin       = mustActionKey(ActionKeyVerbJoin, ActionKeyResourceMember)
	ActionKeyMemberLeave      = mustActionKey(ActionKeyVerbLeave, ActionKeyResourceMember)
	ActionKeyMemberKick       = mustActionKey(ActionKeyVerbKick, ActionKeyResourceMember)
	ActionKeyMemberUpdateRole = mustActionKey(ActionKeyVerbUpdateRole, ActionKeyResourceMember)
	ActionKeyMemberList       = mustActionKey(ActionKeyVerbList, ActionKeyResourceMember)
	ActionKeyMemberGet        = mustActionKey(ActionKeyVerbGet, ActionKeyResourceMember)

	// dm
	ActionKeyDMCreate = mustActionKey(ActionKeyVerbCreate, ActionKeyResourceDM)

	// channel
	ActionKeyChannelCreate    = mustActionKey(ActionKeyVerbCreate, ActionKeyResourceChannel)
	ActionKeyChannelList      = mustActionKey(ActionKeyVerbList, ActionKeyResourceChannel)
	ActionKeyChannelGet       = mustActionKey(ActionKeyVerbGet, ActionKeyResourceChannel)
	ActionKeyChannelUpdate    = mustActionKey(ActionKeyVerbUpdate, ActionKeyResourceChannel)
	ActionKeyChannelArchive   = mustActionKey(ActionKeyVerbArchive, ActionKeyResourceChannel)
	ActionKeyChannelUnarchive = mustActionKey(ActionKeyVerbUnarchive, ActionKeyResourceChannel)
	ActionKeyChannelDelete    = mustActionKey(ActionKeyVerbDelete, ActionKeyResourceChannel)

	// channel_member
	ActionKeyChannelMemberJoin     = mustActionKey(ActionKeyVerbJoin, ActionKeyResourceChannelMember)
	ActionKeyChannelMemberLeave    = mustActionKey(ActionKeyVerbLeave, ActionKeyResourceChannelMember)
	ActionKeyChannelMemberKick     = mustActionKey(ActionKeyVerbKick, ActionKeyResourceChannelMember)
	ActionKeyChannelMemberUpdate   = mustActionKey(ActionKeyVerbUpdate, ActionKeyResourceChannelMember)
	ActionKeyChannelMemberList     = mustActionKey(ActionKeyVerbList, ActionKeyResourceChannelMember)
	ActionKeyChannelMemberMarkRead = mustActionKey(ActionKeyVerbMarkRead, ActionKeyResourceChannelMember)

	// message
	ActionKeyMessageSend   = mustActionKey(ActionKeyVerbSend, ActionKeyResourceMessage)
	ActionKeyMessageReply  = mustActionKey(ActionKeyVerbReply, ActionKeyResourceMessage)
	ActionKeyMessageEdit   = mustActionKey(ActionKeyVerbEdit, ActionKeyResourceMessage)
	ActionKeyMessageDelete = mustActionKey(ActionKeyVerbDelete, ActionKeyResourceMessage)
	ActionKeyMessageList   = mustActionKey(ActionKeyVerbList, ActionKeyResourceMessage)
	ActionKeyMessageGet    = mustActionKey(ActionKeyVerbGet, ActionKeyResourceMessage)
	ActionKeyMessageSearch = mustActionKey(ActionKeyVerbSearch, ActionKeyResourceMessage)
	ActionKeyMessagePin    = mustActionKey(ActionKeyVerbPin, ActionKeyResourceMessage)
	ActionKeyMessageUnpin  = mustActionKey(ActionKeyVerbUnpin, ActionKeyResourceMessage)

	// reaction
	ActionKeyReactionAdd    = mustActionKey(ActionKeyVerbAdd, ActionKeyResourceReaction)
	ActionKeyReactionRemove = mustActionKey(ActionKeyVerbRemove, ActionKeyResourceReaction)
	ActionKeyReactionList   = mustActionKey(ActionKeyVerbList, ActionKeyResourceReaction)

	// attachment
	ActionKeyAttachmentUpload = mustActionKey(ActionKeyVerbUpload, ActionKeyResourceAttachment)
	ActionKeyAttachmentDelete = mustActionKey(ActionKeyVerbDelete, ActionKeyResourceAttachment)
	ActionKeyAttachmentGet    = mustActionKey(ActionKeyVerbGet, ActionKeyResourceAttachment)

	// notification
	ActionKeyNotificationList    = mustActionKey(ActionKeyVerbList, ActionKeyResourceNotification)
	ActionKeyNotificationRead    = mustActionKey(ActionKeyVerbRead, ActionKeyResourceNotification)
	ActionKeyNotificationReadAll = mustActionKey(ActionKeyVerbReadAll, ActionKeyResourceNotification)

	// invite
	ActionKeyInviteCreate = mustActionKey(ActionKeyVerbCreate, ActionKeyResourceInvite)
	ActionKeyInviteRevoke = mustActionKey(ActionKeyVerbRevoke, ActionKeyResourceInvite)
	ActionKeyInviteAccept = mustActionKey(ActionKeyVerbAccept, ActionKeyResourceInvite)
	ActionKeyInviteList   = mustActionKey(ActionKeyVerbList, ActionKeyResourceInvite)

	// status
	ActionKeyStatusSet   = mustActionKey(ActionKeyVerbSet, ActionKeyResourceStatus)
	ActionKeyStatusClear = mustActionKey(ActionKeyVerbClear, ActionKeyResourceStatus)
)
