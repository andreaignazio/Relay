package models

import (
	"encoding/json"
	"gokafka/internal/shared"
	"time"

	"github.com/google/uuid"
)

type EventStore struct {
	EventID       string                   `gorm:"primaryKey;type:uuid;" json:"event_id"`
	AggregateType shared.ActionKeyResource `json:"aggregate_type"`
	AggregateID   uuid.UUID                `gorm:"type:uuid;uniqueIndex:idx_aggregate_version" json:"aggregate_id"`
	Version       int                      `gorm:"uniqueIndex:idx_aggregate_version" json:"version"`
	ActionKey     shared.ActionKey         `gorm:"not null" json:"action_key"`
	Payload       json.RawMessage          `json:"payload"`
	Metadata      shared.MessageMetadata   `gorm:"serializer:json" json:"metadata"`
	CreatedAt     time.Time                `json:"created_at"`
}

type Metadata struct {
	UserID uuid.UUID      `json:"user_id"`
	Extra  map[string]any `json:"extra,omitempty"`
}
