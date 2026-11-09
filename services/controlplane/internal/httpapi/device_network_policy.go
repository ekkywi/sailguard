package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/networkpolicy"
	"github.com/google/uuid"
)

func (a *API) handleGetDeviceNetworkPolicy(w http.ResponseWriter, r *http.Request) {
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

	store := networkpolicy.NewStore(a.pool)
	ep, err := store.EffectiveForDevice(r.Context(), id)
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
				"message": "failed to compute effective network policy",
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
				"message": "failed to encode effective network policy",
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
				"message": "failed to encode effective network policy",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{OK: true, Data: data})
}
