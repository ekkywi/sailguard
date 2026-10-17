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

type patchPolicyRequest struct {
	Mode string `json:"mode"`
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

func (a *API) handlePatchPolicyRule(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(r.PathValue("id"))
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
	ruleID, err := uuid.Parse(r.PathValue("ruleId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid rule id",
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
	rule, err := store.UpdateRule(r.Context(), policyID, ruleID, policy.UpdateRuleInput{
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
				"message": "rule not found",
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
				"message": "failed to update rule",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: policyRuleJSON(*rule),
	})
}

func (a *API) handlePatchPolicy(w http.ResponseWriter, r *http.Request) {
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

	var req patchPolicyRequest
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
	p, err := store.UpdatePolicyMode(r.Context(), id, req.Mode)
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
				"message": "mode must be audit or enforce",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to update policy",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: policyJSON(*p),
	})
}

func (a *API) handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
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
	err = store.DeletePolicy(r.Context(), id)
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
				"message": "failed to delete policy",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"deleted": true,
		},
	})
}

func (a *API) handleDeletePolicyRule(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(r.PathValue("id"))
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
	ruleID, err := uuid.Parse(r.PathValue("ruleId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid rule id",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	err = store.DeleteRule(r.Context(), policyID, ruleID)
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "rule not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to delete rule",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"policy_id": policyID.String(),
			"rule_id":   ruleID.String(),
		},
	})
}
