package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/telemetry"
	"github.com/google/uuid"
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

func parseEventListFilter(r *http.Request) (telemetry.EventListFilter, error) {
	q := r.URL.Query()
	f := telemetry.EventListFilter{
		Q: strings.TrimSpace(q.Get("q")),
	}

	if et := strings.TrimSpace(q.Get("event_type")); et != "" {
		if et != "violation_detected" && et != "violation_blocked" {
			return f, fmt.Errorf("event_type must be violation_detected or violation_blocked")
		}
		f.EventType = et
	}

	if raw := strings.TrimSpace(q.Get("device_id")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return f, fmt.Errorf("invalid device_id")
		}
		f.DeviceID = &id
	}

	if raw := strings.TrimSpace(q.Get("from")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return f, fmt.Errorf("invalid from (use RFC3339)")
		}
		f.From = &t
	}
	if raw := strings.TrimSpace(q.Get("to")); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return f, fmt.Errorf("invalid to (use RFC3339)")
		}
		f.To = &t
	}

	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return f, fmt.Errorf("invalid limit")
		}
		f.Limit = n
	} else {
		f.Limit = 50
	}
	if raw := strings.TrimSpace(q.Get("offset")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return f, fmt.Errorf("invalid offset")
		}
		f.Offset = n
	}

	return f, nil
}

func clampEventLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func (a *API) handleListEvents(w http.ResponseWriter, r *http.Request) {
	f, err := parseEventListFilter(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": err.Error(),
			},
		})
		return
	}

	result, err := telemetry.NewStore(a.pool).ListEvents(r.Context(), f)
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

	items := make([]map[string]any, 0, len(result.Items))
	for _, e := range result.Items {
		items = append(items, deviceEventJSON(e))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"items":  items,
			"total":  result.Total,
			"limit":  clampEventLimit(f.Limit),
			"offset": f.Offset,
		},
	})
}

func (a *API) handleExportEventsCSV(w http.ResponseWriter, r *http.Request) {
	f, err := parseEventListFilter(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": err.Error(),
			},
		})
		return
	}

	items, err := telemetry.NewStore(a.pool).ListEventsForExport(r.Context(), f, 10000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to export events",
			},
		})
		return
	}

	filename := fmt.Sprintf("sailguard-events-%s.csv", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"occurred_at", "received_at", "event_type", "hostname", "device_id",
		"process_name", "process_path", "process_user_name", "process_pid",
		"match_action", "match_rule_type", "match_value", "match_category_code",
		"policy_mode", "policy_version", "client_event_id",
	})
	for _, e := range items {
		pid := ""
		if e.ProcessPID != nil {
			pid = strconv.Itoa(int(*e.ProcessPID))
		}
		polVer := ""
		if e.PolicyVersion != nil {
			polVer = strconv.FormatInt(*e.PolicyVersion, 10)
		}
		str := func(p *string) string {
			if p == nil {
				return ""
			}
			return *p
		}
		_ = cw.Write([]string{
			e.OccurredAt.UTC().Format(time.RFC3339),
			e.ReceivedAt.UTC().Format(time.RFC3339),
			e.EventType,
			e.Hostname,
			e.DeviceID.String(),
			str(e.ProcessName),
			str(e.ProcessPath),
			str(e.ProcessUserName),
			pid,
			str(e.MatchAction),
			str(e.MatchRuleType),
			str(e.MatchValue),
			str(e.MatchCategoryCode),
			str(e.PolicyMode),
			polVer,
			e.ClientEventID,
		})
	}
	cw.Flush()
}
