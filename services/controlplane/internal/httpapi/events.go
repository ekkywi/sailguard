package httpapi

import (
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/telemetry"
)

func deviceEventJSON(e telemetry.DeviceEvent) map[string]any {
	return map[string]any{
		"id":              e.ID.String(),
		"device_id":       e.DeviceID.String(),
		"hostname":        e.Hostname,
		"client_event_id": e.ClientEventID,
		"event_type":      e.EventType,
		"occurred_at":     e.OccurredAt,
		"received_at":     e.ReceivedAt,
		"dedupe_key":      e.DedupeKey,

		"process_pid":          e.ProcessPID,
		"process_name":         e.ProcessName,
		"process_path":         e.ProcessPath,
		"process_hash_sha256":  e.ProcessHashSHA256,
		"process_publisher":    e.ProcessPublisher,
		"process_product_name": e.ProcessProductName,
		"process_user_name":    e.ProcessUserName,

		"match_rule_id":       e.MatchRuleID,
		"match_action":        e.MatchAction,
		"match_rule_type":     e.MatchRuleType,
		"match_value":         e.MatchValue,
		"match_category_code": e.MatchCategoryCode,

		"policy_version": e.PolicyVersion,
		"policy_mode":    e.PolicyMode,
	}
}

func (a *API) handleListEvents(w http.ResponseWriter, r *http.Request) {
	store := telemetry.NewStore(a.pool)
	events, err := store.ListEvents(r.Context(), 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list events",
			},
		})
		return
	}
	items := make([]map[string]any, 0, len(events))
	for _, e := range events {
		items = append(items, deviceEventJSON(e))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": items},
	})
}
