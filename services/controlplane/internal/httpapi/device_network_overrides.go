package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/networkpolicy"
	"github.com/google/uuid"
)

type createDeviceNetworkOverrideRequest struct {
	Action    string  `json:"action"`
	DestType  string  `json:"dest_type"`
	DestValue string  `json:"dest_value"`
	Port      *int    `json:"port"`
	Label     string  `json:"label"`
	Reason    string  `json:"reason"`
	ExpiresAt *string `json:"expires_at"`
}

func deviceNetworkOverrideJSON(o networkpolicy.DeviceOverride) map[string]any {
	now := time.Now().UTC()
	return map[string]any{
		"id":         o.ID.String(),
		"device_id":  o.DeviceID.String(),
		"action":     o.Action,
		"dest_type":  o.DestType,
		"dest_value": o.DestValue,
		"port":       o.Port,
		"label":      o.Label,
		"reason":     o.Reason,
		"expires_at": o.ExpiresAt,
		"expired":    o.Expired(now),
		"created_at": o.CreatedAt,
		"updated_at": o.UpdatedAt,
	}
}

func (a *API) handleListDeviceNetworkOverrides(w http.ResponseWriter, r *http.Request) {
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

	store := networkpolicy.NewStore(a.pool)
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
				"message": "failed to list network overrides",
			},
		})
		return
	}

	out := make([]map[string]any, 0, len(items))
	for _, o := range items {
		out = append(out, deviceNetworkOverrideJSON(o))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"items":           out,
			"include_expired": includeExpired,
		},
	})
}

func (a *API) handleCreateDeviceNetworkOverride(w http.ResponseWriter, r *http.Request) {
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

	var req createDeviceNetworkOverrideRequest
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

	store := networkpolicy.NewStore(a.pool)
	o, err := store.CreateOverride(r.Context(), id, networkpolicy.CreateOverrideInput{
		Action:    strings.TrimSpace(req.Action),
		DestType:  strings.TrimSpace(req.DestType),
		DestValue: strings.TrimSpace(req.DestValue),
		Port:      req.Port,
		Label:     strings.TrimSpace(req.Label),
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
	if errors.Is(err, networkpolicy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid override: allow requires future expires_at; reason and dest fields required",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to create network override",
			},
		})
		return
	}

	a.recordAudit(r, "network_override.create", "device_network_override", o.ID.String(),
		"Created "+o.Action+" network override on device",
		map[string]any{
			"device_id":  o.DeviceID.String(),
			"action":     o.Action,
			"dest_type":  o.DestType,
			"dest_value": o.DestValue,
			"port":       o.Port,
			"label":      o.Label,
			"reason":     o.Reason,
			"expires_at": o.ExpiresAt,
		},
	)

	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: deviceNetworkOverrideJSON(*o),
	})
}

func (a *API) handleDeleteDeviceNetworkOverride(w http.ResponseWriter, r *http.Request) {
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

	store := networkpolicy.NewStore(a.pool)
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
	if errors.Is(err, networkpolicy.ErrNotFound) {
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
				"message": "failed to delete network override",
			},
		})
		return
	}

	a.recordAudit(r, "network_override.delete", "device_network_override", overrideID.String(),
		"Deleted device network override",
		map[string]any{
			"device_id": deviceID.String(),
		},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"deleted": true},
	})
}
