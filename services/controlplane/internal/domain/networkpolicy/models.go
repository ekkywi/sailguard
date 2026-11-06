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

type DeviceOverride struct {
	ID        uuid.UUID
	DeviceID  uuid.UUID
	Action    string
	DestType  string
	DestValue string
	Port      *int
	Label     string
	Reason    string
	ExpiresAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (o DeviceOverride) Expired(now time.Time) bool {
	return o.ExpiresAt != nil && !o.ExpiresAt.After(now)
}

type CreateOverrideInput struct {
	Action    string
	DestType  string
	DestValue string
	Port      *int
	Label     string
	Reason    string
	ExpiresAt *time.Time
}
