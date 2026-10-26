package alerting

import (
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID        uuid.UUID
	Kind      string
	Name      string
	Enabled   bool
	Config    map[string]any
	CreatedAt time.Time
	UpdatedAt time.Time
}

const (
	DeliveryPending   = "pending"
	DeliveryDelivered = "delivered"
	DeliveryFailed    = "failed"
	DeliveryDead      = "dead"
)

type Delivery struct {
	ID          uuid.UUID
	ChannelID   uuid.UUID
	EventID     uuid.UUID
	Status      string
	Attempts    int
	MaxAttempts int
	LastError   *string
	NextRetryAt *time.Time
	DeliveredAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (c Channel) WebhookURL() string {
	if c.Config == nil {
		return ""
	}
	u, _ := c.Config["url"].(string)
	return u
}
