package agentcontract

import "github.com/ekkywi/sailguard/pkgs/policy"

type Envelope[T any] struct {
	OK    bool   `json:"ok"`
	Data  *T     `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
}

type Error struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type EnrollRequest struct {
	SchemaVersion   int    `json:"schema_version"`
	EnrollmentToken string `json:"enrollment_token"`
	Hostname        string `json:"hostname"`
	MachineGUID     string `json:"machine_guid"`
	OSVersion       string `json:"os_version"`
	OSFamily        string `json:"os_family"`
	AgentVersion    string `json:"agent_version"`
}

type EnrollResponse struct {
	DeviceID       string  `json:"device_id"`
	DeviceToken    string  `json:"device_token"`
	TokenExpiresAt *string `json:"token_expires_at"`
	PolicyPollSec  int     `json:"policy_poll_sec"`
	EventFlushSec  int     `json:"event_flush_sec"`
	HeartbeatSec   int     `json:"heartbeat_sec"`
}

type PolicySyncData struct {
	Unchanged bool `json:"unchanged,omitempty"`
	policy.EffectivePolicy
}

type EventBatchRequest struct {
	SchemaVersion int     `json:"schema_version"`
	DeviceID      string  `json:"device_id"`
	SentAt        string  `json:"sent_at"`
	Events        []Event `json:"events"`
}

type Event struct {
	ClientEventID string             `json:"client_event_id"`
	EventType     string             `json:"event_type"`
	OccurredAt    string             `json:"occurred_at"`
	DedupeKey     string             `json:"dedupe_key,omitempty"`
	AgentVersion  string             `json:"agent_version,omitempty"`
	Hostname      string             `json:"hostname,omitempty"`
	OSVersion     string             `json:"os_version,omitempty"`
	PolicyVersion int64              `json:"policy_version,omitempty"`
	PolicyHash    string             `json:"policy_hash,omitempty"`
	Hardware      *HardwareInventory `json:"hardware,omitempty"`
	Metrics       *ResourceMetrics   `json:"metrics,omitempty"`
	PrimaryIP     string             `json:"primary_ip,omitempty"`
	Process       *ProcessInfo       `json:"process,omitempty"`
	Match         *MatchInfo         `json:"match,omitempty"`
	Policy        *PolicyRef         `json:"policy,omitempty"`
	Enforcement   *Enforcement       `json:"enforcement,omitempty"`
	Network       *NetworkDest       `json:"network,omitempty"`
	Error         *AgentError        `json:"error,omitempty"`
}

type HardwareInventory struct {
	CPUModel       string `json:"cpu_model,omitempty"`
	CPUCores       int    `json:"cpu_cores,omitempty"`
	RAMMB          int64  `json:"ram_mb,omitempty"`
	DiskGB         int64  `json:"disk_gb,omitempty"`
	HWManufacturer string `json:"hw_manufacturer,omitempty"`
	HWModel        string `json:"hw_model,omitempty"`
}

type ResourceMetrics struct {
	UptimeSec   int64   `json:"uptime_sec,omitempty"`
	CPUPercent  float64 `json:"cpu_percent,omitempty"`
	RAMUsedMB   int64   `json:"ram_used_mb,omitempty"`
	RAMPercent  float64 `json:"ram_percent,omitempty"`
	DiskPercent float64 `json:"disk_percent,omitempty"`
}

type ProcessInfo struct {
	PID         uint32  `json:"pid"`
	Name        string  `json:"name"`
	Path        string  `json:"path"`
	HashSHA256  string  `json:"hash_sha256,omitempty"`
	Publisher   string  `json:"publisher,omitempty"`
	ProductName string  `json:"product_name,omitempty"`
	UserName    string  `json:"user_name,omitempty"`
	CommandLine *string `json:"command_line"`
}

type MatchInfo struct {
	RuleID       string `json:"rule_id"`
	Action       string `json:"action"`
	RuleType     string `json:"rule_type"`
	Value        string `json:"value"`
	CategoryCode string `json:"category_code,omitempty"`
}

type PolicyRef struct {
	Version int64  `json:"version"`
	Mode    string `json:"mode"`
}

type Enforcement struct {
	Result   string  `json:"result"`
	Attempts int     `json:"attempts"`
	Error    *string `json:"error"`
}

type NetworkDest struct {
	DestType  string `json:"dest_type"`
	DestValue string `json:"dest_value"`
	DestIP    string `json:"dest_ip,omitempty"`
	DestPort  *int   `json:"dest_port,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Direction string `json:"direction,omitempty"`
}

type AgentError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

type EventBatchResponse struct {
	Accepted int             `json:"accepted"`
	Rejected []RejectedEvent `json:"rejected,omitempty"`
	Policy   *PolicyHint     `json:"policy,omitempty"`
}

type RejectedEvent struct {
	ClientEventID string `json:"client_event_id"`
	Code          string `json:"code"`
	Message       string `json:"message"`
}

type PolicyHint struct {
	LatestVersion   int64 `json:"latest_version"`
	ReloadSuggested bool  `json:"reload_suggested"`
}

const (
	EventHeartbeat         = "heartbeat"
	EventViolationDetected = "violation_detected"
	EventViolationBlocked  = "violation_blocked"
	EventNetworkBlocked    = "network_blocked"
	EventAgentError        = "agent_error"
)

type NetworkEffectiveRule struct {
	ID        string `json:"id"`
	Action    string `json:"action,omitempty"`
	DestType  string `json:"dest_type"`
	DestValue string `json:"dest_value"`
	Port      *int   `json:"port,omitempty"`
	Label     string `json:"label,omitempty"`
	Source    string `json:"source,omitempty"`
}

type NetworkPolicySyncData struct {
	Unchanged     bool                   `json:"unchanged,omitempty"`
	SchemaVersion int                    `json:"schema_version"`
	Version       int64                  `json:"version"`
	ComputedAt    string                 `json:"computed_at,omitempty"`
	Mode          string                 `json:"mode"`
	Hash          string                 `json:"hash"`
	SafetyAllow   []NetworkEffectiveRule `json:"safety_allow"`
	Rules         []NetworkEffectiveRule `json:"rules"`
}
