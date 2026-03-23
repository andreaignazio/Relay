package rteventshandler

import (
	"context"
	"encoding/json"
	"fmt"
	"gokafka/internal/domaineventpayloads"
	"gokafka/internal/kafkaclient/producer"
	"gokafka/internal/shared"
)

type EventRoute struct {
	Procucer     *producer.KafkaProducer
	EventBuilder func(ctx context.Context, event shared.Event) (*shared.RTEvent, error)
	PartionFn    func(event shared.Event) shared.RealTimePartitionKey
}

func (h *Handler) GetEventRoutes(event shared.Event) []EventRoute {
	var routes []EventRoute

	switch event.GetActionKey() {
	case shared.ActionKeyWorkspaceCreate:
		var p domaineventpayloads.WorkspaceCreatedPayload
		if err := json.Unmarshal(event.Payload, &p); err != nil {
			fmt.Printf("[RTHandler] failed to unmarshal WorkspaceCreated payload: %v\n", err)
			return routes
		}
		userID := p.UserID.String()
		route := EventRoute{
			Procucer:     h.UserTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewUserRealTimePartitionKey(userID)
			},
		}
		routes = append(routes, route)

	case shared.ActionKeyWorkspaceUpdate, shared.ActionKeyWorkspaceDelete,
		shared.ActionKeyMemberJoin, shared.ActionKeyMemberLeave,
		shared.ActionKeyMemberUpdateRole:
		route := EventRoute{
			Procucer:     h.WorkspaceTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewWorkspaceRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)

	case shared.ActionKeyMemberKick:
		route := EventRoute{
			Procucer:     h.UserTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewUserRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
		route = EventRoute{
			Procucer:     h.WorkspaceTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewWorkspaceRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)

	case shared.ActionKeyChannelCreate:
		route := EventRoute{
			Procucer:     h.WorkspaceTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewWorkspaceRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)

	case shared.ActionKeyChannelUpdate:
		route := EventRoute{
			Procucer:     h.WorkspaceTopicProducer,
			EventBuilder: h.HandleHintEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewWorkspaceRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
		route = EventRoute{
			Procucer:     h.ChannelTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewChannelRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyChannelArchive,
		shared.ActionKeyChannelUnarchive,
		shared.ActionKeyAttachmentDelete:
		route := EventRoute{
			Procucer:     h.WorkspaceTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewWorkspaceRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyChannelMemberJoin,
		shared.ActionKeyChannelMemberLeave,
		shared.ActionKeyChannelMemberUpdate:
		route := EventRoute{
			Procucer:     h.ChannelTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewChannelRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)

	case shared.ActionKeyChannelMemberKick:
		route := EventRoute{
			Procucer:     h.ChannelTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewChannelRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
		route = EventRoute{
			Procucer:     h.UserTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewUserRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyChannelMemberMarkRead:
		route := EventRoute{}
		routes = append(routes, route)
	case shared.ActionKeyMessageSend,
		shared.ActionKeyMessageDelete:
		route := EventRoute{
			Procucer:     h.ChannelTopicProducer,
			EventBuilder: h.HandleRichEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewChannelRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
		route = EventRoute{
			Procucer:     h.WorkspaceTopicProducer,
			EventBuilder: h.HandleHintEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewWorkspaceRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyMessageEdit:
		route := EventRoute{
			Procucer:     h.ChannelTopicProducer,
			EventBuilder: h.HandleRichEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewChannelRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyMessagePin, shared.ActionKeyMessageUnpin:
		route := EventRoute{
			Procucer:     h.ChannelTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewChannelRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyReactionAdd, shared.ActionKeyReactionRemove:
		route := EventRoute{
			Procucer:     h.ChannelTopicProducer,
			EventBuilder: h.HandleRichEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewChannelRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
		route = EventRoute{
			Procucer:     h.WorkspaceTopicProducer,
			EventBuilder: h.HandleHintEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewWorkspaceRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyAttachmentUpload, shared.ActionKeyAttachmentDelete:
		route := EventRoute{
			Procucer:     h.ChannelTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewChannelRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyInviteCreate:
		route := EventRoute{
			Procucer:     h.UserTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewUserRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyInviteAccept:
		route := EventRoute{
			Procucer:     h.UserTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewUserRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
		route = EventRoute{
			Procucer:     h.WorkspaceTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewWorkspaceRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	case shared.ActionKeyStatusSet, shared.ActionKeyStatusClear:
		route := EventRoute{
			Procucer:     h.WorkspaceTopicProducer,
			EventBuilder: h.HandleInvalidateCacheEvent,
			PartionFn: func(e shared.Event) shared.RealTimePartitionKey {
				return shared.NewWorkspaceRealTimePartitionKey(e.GetAggregateID().String())
			},
		}
		routes = append(routes, route)
	default:
		fmt.Printf("[RTHandler]No route defined for ActionKey: %s\n", event.GetActionKey())
	}

	return routes
}

/*func (h *Handler) GetProducerForEvent(event shared.Event) *producer.KafkaProducer {
switch event.GetPartitionKey()*/
