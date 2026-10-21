package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/policy"
	"github.com/google/uuid"
)

type createDeviceOverrideRequest struct {
	Action    string  `json:"action"`
	RuleType  string  `json:"rule_type"`
	Operator  string  `json:"operator"`
	Value     string  `json:"value"`
	Reason    string  `json:"reason"`
	ExpiresAt *string `json:"expires_at"`
}

func deviceOverrideJSON(o policy.DeviceOverride) map[string]any {
	now := time.Now().UTC()
	return map[string]any{
		"id":         o.ID.String(),
		"device_id":  o.DeviceID.String(),
		"action":     o.Action,
		"rule_type":  o.RuleType,
		"operator":   o.Operator,
		"value":      o.Value,
		"reason":     o.Reason,
		"expires_at": o.ExpiresAt,
		"expired":    o.Expired(now),
		"created_at": o.CreatedAt,
		"updated_at": o.UpdatedAt,
	}
}

func (a *API) handleListDeviceOverrides(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid device id",
			},
		})
		return
	}

	includeExpired := r.URL.Query().Get("include_expired") == "1" ||
		r.URL.Query().Get("include_expired") == "true"

	store := policy.NewStore(a.pool)
	items, err := store.ListOverridesByDeviceID(r.Context(), id, includeExpired)
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "device not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list overrides",
			},
		})
		return
	}

	out := make([]map[string]any, 0, len(items))
	for _, o := range items {
		out = append(out, deviceOverrideJSON(o))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"items":           out,
			"include_expired": includeExpired,
		},
	})
}

func (a *API) handleCreateDeviceOverride(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid device id",
			},
		})
		return
	}

	var req createDeviceOverrideRequest
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

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		raw := strings.TrimSpace(*req.ExpiresAt)
		if raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, envelope{
					OK: false,
					Error: map[string]any{
						"code":    "validation_error",
						"message": "expires_at must be RFC3339",
					},
				})
				return
			}
			expiresAt = &t
		}
	}

	store := policy.NewStore(a.pool)
	o, err := store.CreateOverride(r.Context(), id, policy.CreateOverrideInput{
		Action:    strings.TrimSpace(req.Action),
		RuleType:  strings.TrimSpace(req.RuleType),
		Operator:  strings.TrimSpace(req.Operator),
		Value:     strings.TrimSpace(req.Value),
		Reason:    strings.TrimSpace(req.Reason),
		ExpiresAt: expiresAt,
	})
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "device not found",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid override: allow requires future expires_at; reason and match fields required; category not allowed",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to create override",
			},
		})
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: deviceOverrideJSON(*o),
	})
}

func (a *API) handleDeleteDeviceOverride(w http.ResponseWriter, r *http.Request) {
	deviceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid device id",
			},
		})
		return
	}
	overrideID, err := uuid.Parse(r.PathValue("overrideId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid override id",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	err = store.DeleteOverride(r.Context(), deviceID, overrideID)
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "device not found",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "override not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to delete override",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"deleted": true},
	})
}
