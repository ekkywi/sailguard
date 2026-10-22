package httpapi

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/audit"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/identity"
	"github.com/google/uuid"
)

// writeAudit persists an audit row. Failures are logged only.
func (a *API) writeAudit(ctx context.Context, in audit.RecordInput) {
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	if err := audit.NewStore(a.pool).Record(ctx, in); err != nil {
		log.Printf("audit: record %s failed: %v", in.Action, err)
	}
}

// recordAudit resolves the actor from the request context, then writeAudit.
func (a *API) recordAudit(
	r *http.Request,
	action, resourceType, resourceID, summary string,
	meta map[string]any,
) {
	var actorUserID *uuid.UUID
	actorEmail := ""

	if uid, ok := userIDFromCtx(r.Context()); ok {
		actorUserID = &uid
		u, err := identity.NewStore(a.pool).FindByID(r.Context(), uid)
		if err == nil && u != nil {
			actorEmail = u.Email
		}
	}

	a.writeAudit(r.Context(), audit.RecordInput{
		ActorUserID:  actorUserID,
		ActorEmail:   actorEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Summary:      summary,
		Metadata:     meta,
	})
}

// familyToActionPrefix maps UI family chips to action LIKE prefixes.
var auditFamilyPrefix = map[string]string{
	"policy":     "policy",
	"category":   "category",
	"user":       "user",
	"device":     "device",
	"group":      "group",
	"auth":       "auth",
	"package":    "agent_package",
	"override":   "override",
	"enrollment": "enrollment_token",
}

func parseAuditListFilter(r *http.Request) (audit.ListFilter, error) {
	q := r.URL.Query()
	f := audit.ListFilter{
		Actor:        strings.TrimSpace(q.Get("actor")),
		ResourceType: strings.TrimSpace(q.Get("resource_type")),
		Q:            strings.TrimSpace(q.Get("q")),
	}

	if family := strings.TrimSpace(strings.ToLower(q.Get("family"))); family != "" {
		prefix, ok := auditFamilyPrefix[family]
		if !ok {
			return f, fmt.Errorf("invalid family")
		}
		f.ActionPrefix = prefix
	}
	if p := strings.TrimSpace(q.Get("action_prefix")); p != "" {
		f.ActionPrefix = p
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

func auditEntryJSON(e audit.Entry) map[string]any {
	var actorUserID any
	if e.ActorUserID != nil {
		actorUserID = e.ActorUserID.String()
	}
	meta := json.RawMessage(e.Metadata)
	if len(meta) == 0 {
		meta = json.RawMessage(`{}`)
	}
	return map[string]any{
		"id":            e.ID.String(),
		"actor_user_id": actorUserID,
		"actor_email":   e.ActorEmail,
		"action":        e.Action,
		"resource_type": e.ResourceType,
		"resource_id":   e.ResourceID,
		"summary":       e.Summary,
		"metadata":      meta,
		"created_at":    e.CreatedAt,
	}
}

func (a *API) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	f, err := parseAuditListFilter(r)
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

	result, err := audit.NewStore(a.pool).List(r.Context(), f)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list audit logs",
			},
		})
		return
	}

	out := make([]map[string]any, 0, len(result.Items))
	for _, e := range result.Items {
		out = append(out, auditEntryJSON(e))
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"items":  out,
			"total":  result.Total,
			"limit":  clampAuditLimit(f.Limit),
			"offset": f.Offset,
		},
	})
}

func clampAuditLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func (a *API) handleExportAuditLogsCSV(w http.ResponseWriter, r *http.Request) {
	f, err := parseAuditListFilter(r)
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

	items, err := audit.NewStore(a.pool).ListForExport(r.Context(), f, 10000)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to export audit logs",
			},
		})
		return
	}

	filename := fmt.Sprintf("sailguard-audit-%s.csv", time.Now().UTC().Format("20060102-150405"))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{
		"created_at", "actor_email", "actor_user_id", "action",
		"resource_type", "resource_id", "summary", "metadata",
	})
	for _, e := range items {
		actorID := ""
		if e.ActorUserID != nil {
			actorID = e.ActorUserID.String()
		}
		meta := string(e.Metadata)
		if meta == "" {
			meta = "{}"
		}
		_ = cw.Write([]string{
			e.CreatedAt.UTC().Format(time.RFC3339),
			e.ActorEmail,
			actorID,
			e.Action,
			e.ResourceType,
			e.ResourceID,
			e.Summary,
			meta,
		})
	}
	cw.Flush()
}
