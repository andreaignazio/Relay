package rteventshandler

import (
	"fmt"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/shared"

	"github.com/google/uuid"
)

type RouteSpec struct {
	Topic     shared.RealTimeDestination
	EventType shared.RTEventType
}

// routeTable maps each ActionKey to the list of RT routes it produces.
// This is the only place that knows "which topic + which event type" per action.
var routeTable = map[shared.ActionKey][]RouteSpec{
	shared.ActionKeyWorkspaceCreate: {
		{shared.RealTimeDestinationUsers, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyWorkspaceUpdate: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyWorkspaceDelete: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},

	shared.ActionKeyMemberJoin: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyMemberLeave: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyMemberUpdateRole: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyMemberKick: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
		{shared.RealTimeDestinationUsers, shared.RTEventTypeInvalidateCache},
	},

	shared.ActionKeyDMCreate: {
		{shared.RealTimeDestinationUsers, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyChannelCreate: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyChannelUpdate: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeHint},
		{shared.RealTimeDestinationChannels, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyChannelArchive: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyChannelUnarchive: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyChannelDelete: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},

	shared.ActionKeyChannelMemberJoin: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyChannelMemberLeave: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyChannelMemberUpdate: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyChannelMemberKick: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeInvalidateCache},
		{shared.RealTimeDestinationUsers, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyChannelMemberMarkRead: {},

	shared.ActionKeyMessageSend: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeRich},
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeHint},
	},
	shared.ActionKeyMessageEdit: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeRich},
	},
	shared.ActionKeyMessageDelete: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeRich},
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeHint},
	},
	shared.ActionKeyMessagePin: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyMessageUnpin: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeInvalidateCache},
	},

	shared.ActionKeyReactionAdd: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeRich},
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeHint},
	},
	shared.ActionKeyReactionRemove: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeRich},
	},

	shared.ActionKeyAttachmentUpload: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyAttachmentDelete: {
		{shared.RealTimeDestinationChannels, shared.RTEventTypeInvalidateCache},
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},

	shared.ActionKeyInviteCreate: {
		{shared.RealTimeDestinationUsers, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyInviteAccept: {
		{shared.RealTimeDestinationUsers, shared.RTEventTypeInvalidateCache},
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},

	shared.ActionKeyUserUpdate: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyUserDeactivate: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},

	shared.ActionKeyStatusSet: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
	shared.ActionKeyStatusClear: {
		{shared.RealTimeDestinationWorkspaces, shared.RTEventTypeInvalidateCache},
	},
}

// topicToEntity maps each RT destination to the EntityKeys used for partition ID lookup.
var topicToEntity = map[shared.RealTimeDestination]shared.EntityKeys{
	shared.RealTimeDestinationWorkspaces: shared.EntityKeysWorkspace,
	shared.RealTimeDestinationChannels:   shared.EntityKeysChannel,
	shared.RealTimeDestinationUsers:      shared.EntityKeysUser,
	// DM events fan-out to users via EntityKeysUser; EntityKeysDM holds the channel ID
}

func (h *Handler) producerForTopic(topic shared.RealTimeDestination) *producer.KafkaProducer {
	switch topic {
	case shared.RealTimeDestinationWorkspaces:
		return h.WorkspaceTopicProducer
	case shared.RealTimeDestinationChannels:
		return h.ChannelTopicProducer
	case shared.RealTimeDestinationUsers:
		return h.UserTopicProducer
	default:
		panic(fmt.Sprintf("unknown RT destination: %s", topic))
	}
}

func partitionKeyForTopic(topic shared.RealTimeDestination, id uuid.UUID) shared.RealTimePartitionKey {
	switch topic {
	case shared.RealTimeDestinationWorkspaces:
		return shared.NewWorkspaceRealTimePartitionKey(id.String())
	case shared.RealTimeDestinationChannels:
		return shared.NewChannelRealTimePartitionKey(id.String())
	case shared.RealTimeDestinationUsers:
		return shared.NewUserRealTimePartitionKey(id.String())
	default:
		panic(fmt.Sprintf("unknown RT destination: %s", topic))
	}
}
