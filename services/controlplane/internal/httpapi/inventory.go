package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/google/uuid"
)

type createGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type patchGroupRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type patchDeviceRequest struct {
	DisplayName *string `json:"display_name"`
	Status      *string `json:"status"`
	PIC         *string `json:"pic"`
	Location    *string `json:"location"`
	AssetTag    *string `json:"asset_tag"`
	Notes       *string `json:"notes"`
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
		"id":                   d.ID.String(),
		"hostname":             d.Hostname,
		"display_name":         d.DisplayName,
		"os":                   d.OS,
		"os_version":           d.OSVersion,
		"agent_version":        d.AgentVersion,
		"machine_guid":         d.MachineGUID,
		"status":               d.Status,
		"pic":                  d.PIC,
		"location":             d.Location,
		"asset_tag":            d.AssetTag,
		"notes":                d.Notes,
		"cpu_model":            d.CPUModel,
		"cpu_cores":            d.CPUCores,
		"ram_mb":               d.RAMMB,
		"disk_gb":              d.DiskGB,
		"hw_manufacturer":      d.HWManufacturer,
		"hw_model":             d.HWModel,
		"hardware_reported_at": d.HardwareReportedAt,
		"uptime_sec":           d.UptimeSec,
		"cpu_percent":          d.CPUPercent,
		"ram_used_mb":          d.RAMUsedMB,
		"ram_percent":          d.RAMPercent,
		"disk_percent":         d.DiskPercent,
		"metrics_at":           d.MetricsAt,
		"primary_ip":           d.PrimaryIP,
		"ip_reported_at":       d.IPReportedAt,
		"last_seen_at":         d.LastSeenAt,
		"enrolled_at":          d.EnrolledAt,
		"created_at":           d.CreatedAt,
		"updated_at":           d.UpdatedAt,
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

	a.recordAudit(r, "group.create", "group", group.ID.String(),
		"Created group "+group.Name,
		map[string]any{"name": group.Name},
	)

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

func (a *API) handlePatchGroup(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid group id"},
		})
		return
	}

	var req patchGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid JSON body"},
		})
		return
	}
	if req.Name == nil && req.Description == nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "name or description is required"},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	group, err := store.UpdateGroup(r.Context(), id, req.Name, req.Description)
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK:    false,
			Error: map[string]any{"code": "not_found", "message": "group not found"},
		})
		return
	}
	if errors.Is(err, inventory.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "name must not be empty"},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to update group"},
		})
		return
	}

	a.recordAudit(r, "group.update", "group", group.ID.String(),
		"Updated group "+group.Name,
		map[string]any{"name": group.Name},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: groupJSON(*group),
	})
}

func (a *API) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid group id"},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	err = store.DeleteGroup(r.Context(), id)
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
			Error: map[string]any{"code": "internal_error", "message": "failed to delete group"},
		})
		return
	}

	a.recordAudit(r, "group.delete", "group", id.String(),
		"Deleted group",
		map[string]any{"group_id": id.String()},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"deleted": true},
	})
}

func (a *API) handleListDevices(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 50
	offset := 0
	if v := strings.TrimSpace(q.Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if v := strings.TrimSpace(q.Get("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			offset = n
		}
	}

	store := inventory.NewStore(a.pool)
	result, err := store.ListDevices(r.Context(), inventory.DeviceListFilter{
		Status: strings.TrimSpace(q.Get("status")),
		OS:     strings.TrimSpace(q.Get("os")),
		Q:      strings.TrimSpace(q.Get("q")),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to list devices"},
		})
		return
	}

	items := make([]map[string]any, 0, len(result.Items))
	for _, d := range result.Items {
		items = append(items, deviceJSON(d))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"items":  items,
			"total":  result.Total,
			"limit":  limit,
			"offset": offset,
		},
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

func (a *API) handlePatchDevice(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid device id"},
		})
		return
	}

	var req patchDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid JSON body"},
		})
		return
	}
	if req.DisplayName == nil && req.Status == nil &&
		req.PIC == nil && req.Location == nil && req.AssetTag == nil && req.Notes == nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "at least one of display_name, status, pic, location, asset_tag, notes is required",
			},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	device, err := store.UpdateDevice(
		r.Context(), id,
		req.DisplayName, req.Status,
		req.PIC, req.Location, req.AssetTag, req.Notes,
	)
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK:    false,
			Error: map[string]any{"code": "not_found", "message": "device not found"},
		})
		return
	}
	if errors.Is(err, inventory.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "status must be active or disabled"},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to update device"},
		})
		return
	}

	a.recordAudit(r, "device.update", "device", device.ID.String(),
		"Updated device "+device.Hostname,
		map[string]any{
			"hostname":     device.Hostname,
			"display_name": device.DisplayName,
			"status":       device.Status,
			"pic":          device.PIC,
			"location":     device.Location,
			"asset_tag":    device.AssetTag,
		},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: deviceJSON(*device),
	})
}

func (a *API) handleDeleteDevice(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid device id"},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	err = store.DeleteDevice(r.Context(), id)
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
			Error: map[string]any{"code": "internal_error", "message": "failed to delete device"},
		})
		return
	}

	a.recordAudit(r, "device.delete", "device", id.String(),
		"Deleted device",
		map[string]any{
			"device_id": id.String(),
		},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"deleted": true},
	})
}
