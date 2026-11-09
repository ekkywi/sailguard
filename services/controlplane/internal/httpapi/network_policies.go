package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/networkpolicy"
	"github.com/google/uuid"
)

type createNetworkPolicyRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
	Priority    int    `json:"priority"`
}

type patchNetworkPolicyRequest struct {
	Mode string `json:"mode"`
}

type networkPolicyRuleRequest struct {
	Action    string `json:"action"`
	DestType  string `json:"dest_type"`
	DestValue string `json:"dest_value"`
	Port      *int   `json:"port"`
	Label     string `json:"label"`
}

func networkPolicyJSON(p networkpolicy.Policy) map[string]any {
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

func networkPolicyRuleJSON(r networkpolicy.PolicyRule) map[string]any {
	return map[string]any{
		"id":         r.ID.String(),
		"policy_id":  r.PolicyID.String(),
		"action":     r.Action,
		"dest_type":  r.DestType,
		"dest_value": r.DestValue,
		"port":       r.Port,
		"label":      r.Label,
		"enabled":    r.Enabled,
		"created_at": r.CreatedAt,
		"updated_at": r.UpdatedAt,
	}
}

func (a *API) handleListNetworkPolicies(w http.ResponseWriter, r *http.Request) {
	store := networkpolicy.NewStore(a.pool)
	items, err := store.ListPolicies(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list network policies",
			},
		})
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, p := range items {
		out = append(out, networkPolicyJSON(p))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"items": out,
		},
	})
}

func (a *API) handleCreateNetworkPolicy(w http.ResponseWriter, r *http.Request) {
	var req createNetworkPolicyRequest
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
	store := networkpolicy.NewStore(a.pool)
	p, err := store.CreatePolicy(r.Context(), req.Name, req.Description, req.Mode, req.Priority)
	if errors.Is(err, networkpolicy.ErrInvalid) {
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
				"message": "failed to create netowrk policy",
			},
		})
		return
	}
	a.recordAudit(r, "network_policy_created", "network_policy", p.ID.String(),
		"Created network policy "+p.Name,
		map[string]any{
			"name":     p.Name,
			"mode":     p.Mode,
			"priority": p.Priority,
		},
	)
	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: networkPolicyJSON(*p),
	})
}

func (a *API) handleGetNetworkPolicy(w http.ResponseWriter, r *http.Request) {
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
	store := networkpolicy.NewStore(a.pool)
	p, err := store.FindPolicyByID(r.Context(), id)
	if errors.Is(err, networkpolicy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "network policy not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load network policy",
			},
		})
		return
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: networkPolicyJSON(*p),
	})
}

func (a *API) handlePatchNetworkPolicy(w http.ResponseWriter, r *http.Request) {
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
	var req patchNetworkPolicyRequest
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
	store := networkpolicy.NewStore(a.pool)
	p, err := store.UpdatePolicyMode(r.Context(), id, req.Mode)
	if errors.Is(err, networkpolicy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "network policy not found",
			},
		})
		return
	}
	if errors.Is(err, networkpolicy.ErrInvalid) {
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
				"code":  "internal_error",
				"error": "failed to update network policy",
			},
		})
		return
	}
	a.recordAudit(r, "network_policy.update", "network_policy", p.ID.String(),
		"Updated network policy "+p.Name,
		map[string]any{
			"name": p.Name,
			"mode": p.Mode,
		},
	)
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: networkPolicyJSON(*p),
	})
}

func (a *API) handleDeleteNetworkPolicy(w http.ResponseWriter, r *http.Request) {
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
	store := networkpolicy.NewStore(a.pool)
	err = store.DeletePolicy(r.Context(), id)
	if errors.Is(err, networkpolicy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "network policy not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to delete network policy",
			},
		})
		return
	}
	a.recordAudit(r, "network_policy.delete", "network_policy", id.String(),
		"Deleted network policy",
		map[string]any{
			"policy_id": id.String(),
		},
	)
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"deleted": true,
		},
	})
}

func (a *API) handleListNetworkPolicyRules(w http.ResponseWriter, r *http.Request) {
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
	store := networkpolicy.NewStore(a.pool)
	if _, err := store.FindPolicyByID(r.Context(), id); errors.Is(err, networkpolicy.ErrNotFound) {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "network policy not found",
			},
		})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load network policy",
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
				"message": "failed to list network policy rules",
			},
		})
		return
	}
	items := make([]map[string]any, 0, len(rules))
	for _, rule := range rules {
		items = append(items, networkPolicyRuleJSON(rule))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"items": items,
		},
	})
}

func (a *API) handleAddNetworkPolicyRule(w http.ResponseWriter, r *http.Request) {
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
	var req networkPolicyRuleRequest
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
	store := networkpolicy.NewStore(a.pool)
	rule, err := store.AddRule(r.Context(), id, networkpolicy.AddRuleInput{
		Action:    req.Action,
		DestType:  req.DestType,
		DestValue: req.DestValue,
		Port:      req.Port,
		Label:     req.Label,
	})
	if errors.Is(err, networkpolicy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "network policy not found",
			},
		})
		return
	}
	if errors.Is(err, networkpolicy.ErrInvalid) {
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
				"message": "failed to add network policy rule",
			},
		})
		return
	}
	a.recordAudit(r, "network_policy.rule.create", "network_policy", rule.ID.String(),
		"Added network policy rule",
		map[string]any{
			"policy_id":  rule.PolicyID.String(),
			"action":     rule.Action,
			"dest_type":  rule.DestType,
			"dest_value": rule.DestValue,
		},
	)
	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: networkPolicyRuleJSON(*rule),
	})
}

func (a *API) handlePatchNetworkPolicyRule(w http.ResponseWriter, r *http.Request) {
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
	ruleID, err := uuid.Parse(r.PathValue("rule_id"))
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
	var req networkPolicyRuleRequest
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
	store := networkpolicy.NewStore(a.pool)
	rule, err := store.UpdateRule(r.Context(), policyID, ruleID, networkpolicy.UpdateRuleInput{
		Action:    req.Action,
		DestType:  req.DestType,
		DestValue: req.DestValue,
		Port:      req.Port,
		Label:     req.Label,
	})
	if errors.Is(err, networkpolicy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "rule not found",
			},
		})
		return
	}
	if errors.Is(err, networkpolicy.ErrInvalid) {
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
	a.recordAudit(r, "network_policy.rule.update", "network_policy_rule", rule.ID.String(),
		"updated network policy rule",
		map[string]any{
			"policy_id":  rule.PolicyID.String(),
			"action":     rule.Action,
			"dest_type":  rule.DestType,
			"dest_value": rule.DestValue,
		},
	)
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: networkPolicyRuleJSON(*rule),
	})
}

func (a *API) handleDeleteNetworkPolicyRule(w http.ResponseWriter, r *http.Request) {
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
	ruleID, err := uuid.Parse(r.PathValue("rule_id"))
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
	store := networkpolicy.NewStore(a.pool)
	err = store.DeleteRule(r.Context(), policyID, ruleID)
	if errors.Is(err, networkpolicy.ErrNotFound) {
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
	a.recordAudit(r, "network_policy.rule.delete", "network_policy_rule", ruleID.String(),
		"deleted network policy rule",
		map[string]any{
			"policy_id": policyID.String(),
			"rule_id":   ruleID.String(),
		},
	)
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"policy_id": policyID.String(),
			"rule_id":   ruleID.String(),
		},
	})
}
