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
	ID                 uuid.UUID
	Hostname           string
	DisplayName        string
	OS                 string
	OSVersion          string
	AgentVersion       string
	MachineGUID        *string
	Status             string
	PIC                string
	Location           string
	AssetTag           string
	Notes              string
	CPUModel           string
	CPUCores           int
	RAMMB              int64
	DiskGB             int64
	HWManufacturer     string
	HWModel            string
	HardwareReportedAt *time.Time
	UptimeSec          int64
	CPUPercent         float64
	RAMUsedMB          int64
	RAMPercent         float64
	DiskPercent        float64
	MetricsAt          *time.Time
	PrimaryIP          string
	IPReportedAt       *time.Time
	LastSeenAt         *time.Time
	EnrolledAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
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
