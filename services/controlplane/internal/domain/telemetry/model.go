package telemetry

import (
	"time"

	"github.com/google/uuid"
)

type DeviceEvent struct {
	ID            uuid.UUID
	DeviceID      uuid.UUID
	Hostname      string
	ClientEventID string
	EventType     string
	OccurredAt    time.Time
	ReceivedAt    time.Time
	DedupeKey     *string

	ProcessPID         *int32
	ProcessName        *string
	ProcessPath        *string
	ProcessHashSHA256  *string
	ProcessPublisher   *string
	ProcessProductName *string
	ProcessUserName    *string

	MatchRuleID       *string
	MatchAction       *string
	MatchRuleType     *string
	MatchValue        *string
	MatchCategoryCode *string

	PolicyVersion *int64
	PolicyMode    *string
}

type InsertViolationInput struct {
	DeviceID      uuid.UUID
	ClientEventID string
	EventType     string
	OccurredAt    time.Time
	DedupeKey     string

	ProcessPID         uint32
	ProcessName        string
	ProcessPath        string
	ProcessHashSHA256  string
	ProcessPublisher   string
	ProcessProductName string
	ProcessUserName    string

	MatchRuleID       string
	MatchAction       string
	MatchRuleType     string
	MatchValue        string
	MatchCategoryCode string

	PolicyVersion int64
	PolicyMode    string
}
