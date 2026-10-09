package httpapi

import (
	"encoding/json"
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

type addGroupMemberRequest struct {
	DeviceID string `json:"device_id"`
}

func (a *API) handleAddGroupMember(w http.ResponseWriter, r *http.Request) {
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

	var req addGroupMemberRequest
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

	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid device_id",
			},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	err = store.AddGroupMember(r.Context(), groupID, deviceID)
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "group or device not found",
			},
		})
		return
	}
	if errors.Is(err, inventory.ErrAlreadyMember) {
		writeJSON(w, http.StatusConflict, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "already_member",
				"message": "device is already in this group",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to add group member",
			},
		})
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		OK: true,
		Data: map[string]any{
			"group_id":  groupID.String(),
			"device_id": deviceID.String(),
		},
	})
}

func (a *API) handleRemoveGroupMember(w http.ResponseWriter, r *http.Request) {
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

	deviceID, err := uuid.Parse(r.PathValue("deviceId"))
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

	store := inventory.NewStore(a.pool)
	err = store.RemoveGroupMember(r.Context(), groupID, deviceID)
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "membership not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to remove group member",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"group_id":  groupID.String(),
			"device_id": deviceID.String(),
		},
	})
}
