package shared

type EntityKeys = ActionKeyResource

const (
	EntityKeysWorkspace = ActionKeyResourceWorkspace
	EntityKeysChannel   = ActionKeyResourceChannel
	EntityKeysUser      = ActionKeyResourceUser
)

type ServicesKeys string

const (
	ServicesKeysWorkspaces ServicesKeys = "workspaces-service"
	ServicesKeysChannels   ServicesKeys = "channels-service"
	ServicesKeysMessages   ServicesKeys = "messages-service"
	ServicesKeysUsers      ServicesKeys = "users-service"
)
