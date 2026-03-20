package shared

import "strings"

type PartitionKeyKind string

const (
	PartitionKeyKindEvent   PartitionKeyKind = "evt"
	PartitionKeyKindCommand PartitionKeyKind = "cmd"
)

type PartitionKey interface {
	Bytes() []byte
	String() string
	ID() string
}

type EventPartitionKey struct {
	Kind         PartitionKeyKind
	ParentEntity EntityKeys
	ParentID     string
}

type CommandPartitionKey struct {
	Kind         PartitionKeyKind
	ServicesKeys ServicesKeys
	EntityID     string
}

func (epk EventPartitionKey) String() string {
	return string(epk.Kind) + ":" + string(epk.ParentEntity) + ":" + epk.ParentID
}

func (epk EventPartitionKey) Bytes() []byte {
	return []byte(epk.String())
}

func (epk EventPartitionKey) ID() string {
	return epk.ParentID
}

func (cpk CommandPartitionKey) String() string {
	return string(cpk.Kind) + ":" + string(cpk.ServicesKeys) + ":" + cpk.EntityID
}

func (cpk CommandPartitionKey) Bytes() []byte {
	return []byte(cpk.String())
}
func (cpk CommandPartitionKey) ID() string {
	return cpk.EntityID
}

func NewWorkspaceEventPartitionKey(workspaceID string) EventPartitionKey {
	return EventPartitionKey{
		Kind:         PartitionKeyKindEvent,
		ParentEntity: EntityKeysWorkspace,
		ParentID:     workspaceID,
	}
}

func NewWorkspaceCommandPartitionKey(workspaceID string) CommandPartitionKey {
	return CommandPartitionKey{
		Kind:         PartitionKeyKindCommand,
		ServicesKeys: ServicesKeysWorkspaces,
		EntityID:     workspaceID,
	}
}

func ParsePartitionKey(keyStr string) PartitionKey {
	parts := strings.Split(keyStr, ":")
	if len(parts) < 3 {
		panic("invalid partition key format: " + keyStr)
	}
	switch PartitionKeyKind(parts[0]) {
	case PartitionKeyKindEvent:
		return parseEventPartitionKey(parts)
	case PartitionKeyKindCommand:
		return parseCommandPartitionKey(parts)
	default:
		panic("unknown partition key kind: " + parts[0])
	}
}

func parseEventPartitionKey(parts []string) EventPartitionKey {

	prefix, id := parts[1], parts[2]
	switch EntityKeys(prefix) {
	case EntityKeysWorkspace:
		return NewWorkspaceEventPartitionKey(id)
	default:
		panic("unknown partition key prefix: " + prefix)
	}
}

func parseCommandPartitionKey(parts []string) PartitionKey {

	prefix, id := parts[1], parts[2]
	switch ServicesKeys(prefix) {
	case ServicesKeysWorkspaces:
		return NewWorkspaceCommandPartitionKey(id)

	default:
		panic("unknown command partition key prefix: " + prefix)
	}
}
