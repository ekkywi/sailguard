package audit

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Entry struct {
	ID           uuid.UUID
	ActorUserID  *uuid.UUID
	ActorEmail   string
	Action       string
	ResourceType string
	ResourceID   string
	Summary      string
	Metadata     json.RawMessage
	CreatedAt    time.Time
}

type RecordInput struct {
	ActorUserID  *uuid.UUID
	ActorEmail   string
	Action       string
	ResourceType string
	ResourceID   string
	Summary      string
	Metadata     map[string]any
}
