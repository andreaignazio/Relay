package rteventshandler

import "gokafka/internal/shared"

func (h *Handler) IsRichEvent(event shared.Event) bool {
	switch event.GetActionKey() {
	case shared.ActionKeyMessageSend,
		shared.ActionKeyMessageEdit,
		shared.ActionKeyMessageDelete,
		shared.ActionKeyReactionAdd,
		shared.ActionKeyReactionRemove:
		return true
	default:
		return false
	}
}

func (h *Handler) IsInvalidateCacheEvent(event shared.Event) bool {
	switch event.GetActionKey() {
	case shared.ActionKeyUserUpdate,
		shared.ActionKeyUserDeactivate,

		// Workspace-related events
		shared.ActionKeyWorkspaceCreate,
		shared.ActionKeyWorkspaceUpdate,
		shared.ActionKeyWorkspaceDelete,

		// Membership-related events
		shared.ActionKeyMemberJoin,
		shared.ActionKeyMemberLeave,
		shared.ActionKeyMemberKick,
		shared.ActionKeyMemberUpdateRole,

		// Channel-related events
		shared.ActionKeyChannelCreate,
		shared.ActionKeyChannelUpdate,
		shared.ActionKeyChannelDelete,
		shared.ActionKeyChannelArchive,
		shared.ActionKeyChannelUnarchive,

		// Channel membership-related events
		shared.ActionKeyChannelMemberJoin,
		shared.ActionKeyChannelMemberLeave,
		shared.ActionKeyChannelMemberKick,
		shared.ActionKeyChannelMemberUpdate,
		shared.ActionKeyChannelMemberMarkRead,

		// Message-related events
		shared.ActionKeyMessagePin,
		shared.ActionKeyMessageUnpin,

		//Attachment-related events
		shared.ActionKeyAttachmentUpload,
		shared.ActionKeyAttachmentDelete,

		//Invite-related events
		shared.ActionKeyInviteCreate,
		shared.ActionKeyInviteRevoke,
		shared.ActionKeyInviteAccept,

		//Status-related events
		shared.ActionKeyStatusSet,
		shared.ActionKeyStatusClear:
		return true
	default:
		return false
	}
}

func (h *Handler) IsHintEvent(event shared.Event) bool {
	switch event.GetActionKey() {
	case shared.ActionKeyWorkspaceUpdate,
		shared.ActionKeyMessageSend,
		shared.ActionKeyMessageDelete,
		shared.ActionKeyReactionAdd:
		return true
	default:
		return false
	}
}
