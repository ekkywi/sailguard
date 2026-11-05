package networkpolicy

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
	ID        uuid.UUID
	PolicyID  uuid.UUID
	Action    string
	DestType  string
	DestValue string
	Port      *int
	Label     string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AddRuleInput struct {
	Action    string
	DestType  string
	DestValue string
	Port      *int
	Label     string
}

type UpdateRuleInput struct {
	Action    string
	DestType  string
	DestValue string
	Port      *int
	Label     string
}
