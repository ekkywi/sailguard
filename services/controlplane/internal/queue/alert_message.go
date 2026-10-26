package queue

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type AlertMessage struct {
	DeliveryID string `json:"delivery_id"`
	ChannelID  string `json:"channel_id"`
	EventID    string `json:"event_id"`
}

func EncodeAlertValues(msg AlertMessage) (map[string]any, error) {
	msg.DeliveryID = strings.TrimSpace(msg.DeliveryID)
	msg.ChannelID = strings.TrimSpace(msg.ChannelID)
	msg.EventID = strings.TrimSpace(msg.EventID)
	if msg.DeliveryID == "" || msg.ChannelID == "" || msg.EventID == "" {
		return nil, fmt.Errorf("delivery_id, channel_id, and event_id are required")
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal alert message: %w", err)
	}
	return map[string]any{
		FieldPayload: string(raw),
	}, nil
}

func DecodeAlertMessage(values map[string]interface{}) (AlertMessage, error) {
	var msg AlertMessage
	if values == nil {
		return msg, fmt.Errorf("empty stream values")
	}
	raw, ok := values[FieldPayload]
	if !ok {
		return msg, fmt.Errorf("missing field %q", FieldPayload)
	}
	s, ok := raw.(string)
	if !ok {
		return msg, fmt.Errorf("field %q must be a string", FieldPayload)
	}
	if err := json.Unmarshal([]byte(s), &msg); err != nil {
		return msg, fmt.Errorf("unmarshal payload: %w", err)
	}
	return msg, nil
}

func (m AlertMessage) DeliveryUUID() (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(m.DeliveryID))
	if err != nil {
		return uuid.Nil, fmt.Errorf("delivery_id: %w", err)
	}
	return id, nil
}

func (m AlertMessage) ChannelUUID() (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(m.ChannelID))
	if err != nil {
		return uuid.Nil, fmt.Errorf("channel_id: %w", err)
	}
	return id, nil
}

func (m AlertMessage) EventUUID() (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(m.EventID))
	if err != nil {
		return uuid.Nil, fmt.Errorf("event_id: %w", err)
	}
	return id, nil
}
