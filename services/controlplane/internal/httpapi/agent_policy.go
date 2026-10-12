package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/policy"
)

func (a *API) handleAgentPolicy(w http.ResponseWriter, r *http.Request) {
	deviceID, ok := deviceIDFromCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "missing device auth context",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	ep, err := store.EffectiveForDevice(r.Context(), deviceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to compute effective policy",
			},
		})
		return
	}

	clientHash := strings.TrimSpace(r.URL.Query().Get("hash"))
	if clientHash != "" && clientHash == ep.Hash {
		writeJSON(w, http.StatusOK, envelope{
			OK: true,
			Data: map[string]any{
				"unchanged": true,
			},
		})
		return
	}

	raw, err := json.Marshal(ep)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to encode policy",
			},
		})
		return
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to encode policy",
			},
		})
		return
	}
	data["unchanged"] = false

	writeJSON(w, http.StatusOK, envelope{OK: true, Data: data})
}
