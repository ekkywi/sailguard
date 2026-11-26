package queue

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const FieldPayload = "payload"

type ViolationMessage struct {
	DeviceID      string `json:"device_id"`
	ClientEventID string `json:"client_event_id"`
	EventType     string `json:"event_type"`
	OccurredAt    string `json:"occurred_at"`
	DedupeKey     string `json:"dedupe_key,omitempty"`

	ProcessPID         uint32 `json:"process_pid,omitempty"`
	ProcessName        string `json:"process_name"`
	ProcessPath        string `json:"process_path,omitempty"`
	ProcessHashSHA256  string `json:"process_hash_sha256,omitempty"`
	ProcessPublisher   string `json:"process_publisher,omitempty"`
	ProcessProductName string `json:"process_product_name,omitempty"`
	ProcessUserName    string `json:"process_user_name,omitempty"`

	MatchRuleID       string `json:"match_rule_id,omitempty"`
	MatchAction       string `json:"match_action,omitempty"`
	MatchRuleType     string `json:"match_rule_type,omitempty"`
	MatchValue        string `json:"match_value,omitempty"`
	MatchCategoryCode string `json:"match_category_code,omitempty"`

	PolicyVersion int64  `json:"policy_version,omitempty"`
	PolicyMode    string `json:"policy_mode,omitempty"`

	NetworkDestType  string `json:"network_dest_type,omitempty"`
	NetworkDestValue string `json:"network_dest_value,omitempty"`
	NetworkDestIP    string `json:"network_dest_ip,omitempty"`
	NetworkDestPort  *int   `json:"network_dest_port,omitempty"`
	NetworkProtocol  string `json:"network_protocol,omitempty"`
	NetworkDirection string `json:"network_direction,omitempty"`
}

func EncodeViolationValues(msg ViolationMessage) (map[string]any, error) {
	msg.DeviceID = strings.TrimSpace(msg.DeviceID)
	msg.ClientEventID = strings.TrimSpace(msg.ClientEventID)
	msg.EventType = strings.TrimSpace(msg.EventType)
	msg.ProcessName = strings.TrimSpace(msg.ProcessName)
	msg.NetworkDestType = strings.TrimSpace(msg.NetworkDestType)
	msg.NetworkDestValue = strings.TrimSpace(msg.NetworkDestValue)
	msg.NetworkDestIP = strings.TrimSpace(msg.NetworkDestIP)
	msg.NetworkProtocol = strings.TrimSpace(msg.NetworkProtocol)
	msg.NetworkDirection = strings.TrimSpace(msg.NetworkDirection)

	if msg.DeviceID == "" || msg.ClientEventID == "" || msg.EventType == "" {
		return nil, fmt.Errorf("device_id, client_event_id, and event_type are required")
	}

	switch msg.EventType {
	case "violation_detected", "violation_blocked":
		if msg.ProcessName == "" {
			return nil, fmt.Errorf("process_name is required for %s", msg.EventType)
		}
	case "network_blocked":
		if msg.NetworkDestValue == "" && msg.NetworkDestIP == "" {
			return nil, fmt.Errorf("network dest_value or dest_ip is required for network_blocked")
		}
	default:
		return nil, fmt.Errorf("unsupported event_type %q", msg.EventType)
	}

	if msg.OccurredAt == "" {
		msg.OccurredAt = time.Now().UTC().Format(time.RFC3339)
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal violation message: %w", err)
	}
	return map[string]any{
		FieldPayload: string(raw),
	}, nil
}

func DecodeViolationMessage(values map[string]interface{}) (ViolationMessage, error) {
	var msg ViolationMessage
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

func (m ViolationMessage) DeviceUUID() (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(m.DeviceID))
	if err != nil {
		return uuid.Nil, fmt.Errorf("device_id: %w", err)
	}
	return id, nil
}

func (m ViolationMessage) OccurredAtTime() time.Time {
	ts := strings.TrimSpace(m.OccurredAt)
	if ts == "" {
		return time.Now().UTC()
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Now().UTC()
	}
	return t.UTC()
}
