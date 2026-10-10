package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/policy"
	"github.com/google/uuid"
)

type createPolicyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	Priority    int    `json:"priority"`
}

type addPolicyRuleRequest struct {
	Action       string `json:"action"`
	RuleType     string `json:"rule_type"`
	Operator     string `json:"operator"`
	Value        string `json:"value"`
	CategoryCode string `json:"category_code"`
}

func policyJSON(p policy.Policy) map[string]any {
	return map[string]any{
		"id":           p.ID.String(),
		"name":         p.Name,
		"description":  p.Description,
		"mode":         p.Mode,
		"priority":     p.Priority,
		"version":      p.Version,
		"published_at": p.PublishedAt,
		"created_at":   p.CreatedAt,
		"updated_at":   p.UpdatedAt,
	}
}

func policyRuleJSON(r policy.PolicyRule) map[string]any {
	return map[string]any{
		"id":            r.ID.String(),
		"policy_id":     r.PolicyID.String(),
		"action":        r.Action,
		"rule_type":     r.RuleType,
		"operator":      r.Operator,
		"value":         r.Value,
		"category_code": r.CategoryCode,
		"enabled":       r.Enabled,
		"created_at":    r.CreatedAt,
		"updated_at":    r.UpdatedAt,
	}
}

func (a *API) handleListPolicies(w http.ResponseWriter, r *http.Request) {
	store := policy.NewStore(a.pool)
	items, err := store.ListPolicies(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list policies",
			},
		})
		return
	}

	out := make([]map[string]any, 0, len(items))
	for _, p := range items {
		out = append(out, policyJSON(p))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": out},
	})
}

func (a *API) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req createPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid JSON body",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	p, err := store.CreatePolicy(r.Context(), req.Name, req.Description, req.Mode, req.Priority)
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid policy fields (name/mode/priority)",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to create policy",
			},
		})
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: policyJSON(*p),
	})
}

func (a *API) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid policy id",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	p, err := store.FindPolicyByID(r.Context(), id)
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "policy not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load policy",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: policyJSON(*p),
	})
}

func (a *API) handleListPolicyRules(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid policy id",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	if _, err := store.FindPolicyByID(r.Context(), id); errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "policy not found",
			},
		})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load policy",
			},
		})
		return
	}

	rules, err := store.ListRulesByPolicyID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list policy rules",
			},
		})
		return
	}

	items := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		items = append(items, policyRuleJSON(rule))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": items},
	})
}

func (a *API) handleAddPolicyRule(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid policy id",
			},
		})
		return
	}

	var req addPolicyRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid JSON body",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	rule, err := store.AddRule(r.Context(), id, policy.AddRuleInput{
		Action:       strings.TrimSpace(req.Action),
		RuleType:     strings.TrimSpace(req.RuleType),
		Operator:     strings.TrimSpace(req.Operator),
		Value:        strings.TrimSpace(req.Value),
		CategoryCode: strings.TrimSpace(req.CategoryCode),
	})
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "policy not found",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid rule fields",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to add policy rule",
			},
		})
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: policyRuleJSON(*rule),
	})
}
