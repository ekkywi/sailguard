package inventory

import (
	"time"

	"github.com/google/uuid"
)

type DeviceGroup struct {
	ID          uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Device struct {
	ID           uuid.UUID
	Hostname     string
	DisplayName  string
	OS           string
	OSVersion    string
	AgentVersion string
	MachineGUID  *string
	Status       string
	LastSeenAt   *time.Time
	EnrolledAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type EnrollmentToken struct {
	ID        uuid.UUID
	Label     string
	TokenHash string
	MaxUses   int
	UseCount  int
	ExpiresAt *time.Time
	RevokedAt *time.Time
	CreatedBy *uuid.UUID
	CreatedAt time.Time
}
