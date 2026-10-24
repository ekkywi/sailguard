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

func (c Channel) WebhookURL() string {
	if c.Config == nil {
		return ""
	}
	u, _ := c.Config["url"].(string)
	return u
}