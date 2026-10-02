package identity

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	Name         string
	PasswordHash string
	AuthProvider string
	IsActive     bool
	LastLoginAt  *time.Time
}
