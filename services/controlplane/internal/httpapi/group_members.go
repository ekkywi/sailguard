package httpapi

import (
	"errors"
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/google/uuid"
)

func (a *API) handleListGroupMembers(w http.ResponseWriter, r *http.Request) {
	groupID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid group id",
			},
		})
		return
	}

	store := inventory.NewStore(a.pool)

	if _, err := store.FindGroupByID(r.Context(), groupID); err != nil {
		if errors.Is(err, inventory.ErrNotFound) {
			writeJSON(w, http.StatusNotFound, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "not_found",
					"message": "group not found",
				},
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load group",
			},
		})
		return
	}

	devices, err := store.ListGroupMembers(r.Context(), groupID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list group members",
			},
		})
		return
	}

	items := make([]map[string]any, 0, len(devices))
	for _, d := range devices {
		items = append(items, deviceJSON(d))
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"items": items,
		},
	})
}
