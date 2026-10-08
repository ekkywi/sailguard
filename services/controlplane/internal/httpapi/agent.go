package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekkywi/sailguard/pkgs/agentcontract"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
)

func (a *API) handleAgentEnroll(w http.ResponseWriter, r *http.Request) {
	var req agentcontract.EnrollRequest
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

	token := strings.TrimSpace(req.EnrollmentToken)
	hostname := strings.TrimSpace(req.Hostname)
	if token == "" || hostname == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "enrollment_token and hostname are required",
			},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	device, deviceToken, err := store.EnrollDevice(
		r.Context(),
		token,
		hostname,
		req.OSFamily,
		req.OSVersion,
		req.AgentVersion,
		req.MachineGUID,
	)

	if err != nil {
		switch {
		case errors.Is(err, inventory.ErrEnrollmentInvalid):
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "enrollment_invalid",
					"message": "enrollment token is invalid or expired",
				},
			})
		case errors.Is(err, inventory.ErrEnrollmentExhausted):
			writeJSON(w, http.StatusConflict, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "enrollment_exhausted",
					"message": "enrollment token has no uses remaining",
				},
			})
		case errors.Is(err, inventory.ErrDeviceExists):
			writeJSON(w, http.StatusConflict, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "device_exists",
					"message": "device already enrolled",
				},
			})
		default:
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "internal_error",
					"message": "enrollment failed",
				},
			})
		}
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		OK: true,
		Data: map[string]any{
			"device_id":        device.ID.String(),
			"device_token":     deviceToken,
			"token_expires_at": nil,
			"policy_poll_sec":  60,
			"event_flush_sec":  30,
			"heartbeat_sec":    60,
		},
	})
}
