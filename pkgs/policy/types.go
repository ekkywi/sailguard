package policy

// SchemaVersion is the shared policy/event contract version for agents.
const SchemaVersion = 1

type Mode string

const (
	ModeAudit   Mode = "audit"
	ModeEnforce Mode = "enforce"
)

type Action string

const (
	ActionBlock Action = "block"
	ActionAllow Action = "allow"
)

type RuleType string

const (
	RuleTypeProcessName RuleType = "process_name"
	RuleTypePath        RuleType = "path"
	RuleTypeHashSHA256  RuleType = "hash_sha256"
	RuleTypePublisher   RuleType = "publisher"
	RuleTypeProductName RuleType = "product_name"
	RuleTypeCategory    RuleType = "category" // expanded server-side before agent sync
)

type Operator string

const (
	OperatorEquals   Operator = "equals"
	OperatorPrefix   Operator = "prefix"
	OperatorContains Operator = "contains"
)

type Scope string

const (
	ScopeOrg      Scope = "org"
	ScopeGroup    Scope = "group"
	ScopeDevice   Scope = "device"
	ScopeOverride Scope = "override"
)

// ScopeWeight used during merge. Higher wins.
const (
	WeightOrg      = 100
	WeightGroup    = 200
	WeightDevice   = 300
	WeightOverride = 400
)

// ProcessAttrs is what the agent extracts from a running process for matching.
type ProcessAttrs struct {
	Name        string
	Path        string
	HashSHA256  string
	Publisher   string
	ProductName string
}

// Candidate is a rule atom before/during merge (after category expand).
type Candidate struct {
	ID           string
	PolicyID     string
	Action       Action
	RuleType     RuleType
	Operator     Operator
	Value        string
	CategoryCode string
	Source       string // policy | category | override
	Scope        Scope
	ScopeWeight  int
	PolicyMode   Mode
	PolicyPrio   int
	UpdatedUnix  int64
	ExpiresUnix  int64 // 0 = no expiry; used for overrides
	Reason       string
}

// EffectiveRule is a flattened rule sent to agents (no category left).
type EffectiveRule struct {
	ID           string         `json:"id"`
	Action       Action         `json:"action"`
	RuleType     RuleType       `json:"rule_type"`
	Operator     Operator       `json:"operator"`
	Value        string         `json:"value"`
	ValueMeta    map[string]any `json:"value_meta,omitempty"`
	CategoryCode string         `json:"category_code,omitempty"`
}

// EffectivePolicy is the document agents cache and enforce.
type EffectivePolicy struct {
	SchemaVersion int             `json:"schema_version"`
	Version       int64           `json:"version"`
	ComputedAt    string          `json:"computed_at,omitempty"`
	Mode          Mode            `json:"mode"`
	Hash          string          `json:"hash"`
	Rules         []EffectiveRule `json:"rules"`
}
