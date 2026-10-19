package httpapi

import (
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/telemetry"
)

func (a *API) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	store := telemetry.NewStore(a.pool)
	sum, err := store.Summary(r.Context(), telemetry.DefaultOnlineWindowSec)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load dashboard summary",
			},
		})
		return
	}

	recent := make([]map[string]any, 0, len(sum.RecentViolations))
	for _, v := range sum.RecentViolations {
		recent = append(recent, map[string]any{
			"id":           v.ID,
			"device_id":    v.DeviceID,
			"hostname":     v.Hostname,
			"event_type":   v.EventType,
			"process_name": v.ProcessName,
			"occurred_at":  v.OccurredAt,
		})
	}

	topProcesses := make([]map[string]any, 0, len(sum.TopProcesses))
	for _, p := range sum.TopProcesses {
		topProcesses = append(topProcesses, map[string]any{
			"process_name": p.ProcessName,
			"count":        p.Count,
		})
	}

	topHosts := make([]map[string]any, 0, len(sum.TopHosts))
	for _, h := range sum.TopHosts {
		topHosts = append(topHosts, map[string]any{
			"device_id": h.DeviceID,
			"hostname":  h.Hostname,
			"count":     h.Count,
		})
	}

	staleDevices := make([]map[string]any, 0, len(sum.StaleDevices))
	for _, d := range sum.StaleDevices {
		staleDevices = append(staleDevices, map[string]any{
			"device_id":    d.DeviceID,
			"hostname":     d.Hostname,
			"last_seen_at": d.LastSeenAt,
		})
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"generated_at":      sum.GeneratedAt,
			"online_window_sec": sum.OnlineWindowSec,
			"devices": map[string]any{
				"total":    sum.Devices.Total,
				"active":   sum.Devices.Active,
				"disabled": sum.Devices.Disabled,
				"pending":  sum.Devices.Pending,
				"online":   sum.Devices.Online,
				"stale":    sum.Devices.Stale,
			},
			"violations_24h": map[string]any{
				"detected": sum.Violations24h.Detected,
				"blocked":  sum.Violations24h.Blocked,
			},
			"recent_violations": recent,
			"top_processes":     topProcesses,
			"top_hosts":         topHosts,
			"stale_devices":     staleDevices,
		},
	})
}
