package policy

import (
	"time"

	"github.com/google/uuid"
)

type Policy struct {
	ID          uuid.UUID
	Name        string
	Description string
	Mode        string // audit | enforce
	Priority    int
	Version     int64
	PublishedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PolicyRule struct {
	ID           uuid.UUID
	PolicyID     uuid.UUID
	Action       string // allow | block
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
	CategoryCode string // required when rule_type=category
}
