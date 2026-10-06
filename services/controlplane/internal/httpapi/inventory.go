package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/google/uuid"
)

type createGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func groupJSON(g inventory.DeviceGroup) map[string]any {
	return map[string]any{
		"id":          g.ID.String(),
		"name":        g.Name,
		"description": g.Description,
		"created_at":  g.CreatedAt,
		"updated_at":  g.UpdatedAt,
	}
}

func deviceJSON(d inventory.Device) map[string]any {
	return map[string]any{
		"id":            d.ID.String(),
		"hostname":      d.Hostname,
		"display_name":  d.DisplayName,
		"os":            d.OS,
		"os_version":    d.OSVersion,
		"agent_version": d.AgentVersion,
		"machine_guid":  d.MachineGUID,
		"status":        d.Status,
		"last_seen_at":  d.LastSeenAt,
		"enrolled_at":   d.EnrolledAt,
		"created_at":    d.CreatedAt,
		"updated_at":    d.UpdatedAt,
	}
}

func (a *API) handleListGroups(w http.ResponseWriter, r *http.Request) {
	store := inventory.NewStore(a.pool)
	groups, err := store.ListGroups(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to list groups"},
		})
		return
	}

	items := make([]map[string]any, 0, len(groups))
	for _, g := range groups {
		items = append(items, groupJSON(g))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": items},
	})
}

func (a *API) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid JSON body"},
		})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "name is required"},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	group, err := store.CreateGroup(r.Context(), req.Name, req.Description)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to create group"},
		})
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: groupJSON(*group),
	})
}

func (a *API) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid group id"},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	group, err := store.FindGroupByID(r.Context(), id)
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK:    false,
			Error: map[string]any{"code": "not_found", "message": "group not found"},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to load group"},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: groupJSON(*group),
	})
}

func (a *API) handleListDevices(w http.ResponseWriter, r *http.Request) {
	store := inventory.NewStore(a.pool)
	devices, err := store.ListDevices(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to list devices"},
		})
		return
	}

	items := make([]map[string]any, 0, len(devices))
	for _, d := range devices {
		items = append(items, deviceJSON(d))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": items},
	})
}

func (a *API) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid device id"},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	device, err := store.FindDeviceByID(r.Context(), id)
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK:    false,
			Error: map[string]any{"code": "not_found", "message": "device not found"},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to load device"},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: deviceJSON(*device),
	})
}
