package policy

import (
	"time"

	"github.com/google/uuid"
)

type Policy struct {
	ID          uuid.UUID
	Name        string
	Description string
	Mode        string
	Priority    int
	Version     int64
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyRule struct {
	ID           uuid.UUID
	PolicyID     uuid.UUID
	Action       string
	RuleType     string
	Operator     string
	Value        string
	CategoryCode *string
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AddRuleInput struct {
	Action       string
	RuleType     string
	Operator     string
	Value        string
	CategoryCode string
}

type PolicyAssignment struct {
	ID        uuid.UUID
	PolicyID  uuid.UUID
	Scope     string
	GroupID   *uuid.UUID
	DeviceID  *uuid.UUID
	CreatedAt time.Time
}

type AddAssignmentInput struct {
	Scope    string
	GroupID  string
	DeviceID string
}
