package shared

type ActionKey string

const (
	ActionKeyWorkspaceCreate ActionKey = "workspace.create"
	ActionKeyWorkspaceUpdate ActionKey = "workspace.update"
	ActionKeyWorkspaceDelete ActionKey = "workspace.delete"
)
