package httpapi

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/audit"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/identity"
	"github.com/google/uuid"
)

// recordAudit writes an admin audit row. Failures are logged only —
// they must not fail the user-facing action.
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

	if meta == nil {
		meta = map[string]any{}
	}

	err := audit.NewStore(a.pool).Record(r.Context(), audit.RecordInput{
		ActorUserID:  actorUserID,
		ActorEmail:   actorEmail,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Summary:      summary,
		Metadata:     meta,
	})
	if err != nil {
		log.Printf("audit: record %s failed: %v", action, err)
	}
}

func (a *API) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	items, err := audit.NewStore(a.pool).List(r.Context(), 50)
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

	out := make([]map[string]any, 0, len(items))
	for _, e := range items {
		var actorUserID any
		if e.ActorUserID != nil {
			actorUserID = e.ActorUserID.String()
		}
		meta := json.RawMessage(e.Metadata)
		if len(meta) == 0 {
			meta = json.RawMessage(`{}`)
		}
		out = append(out, map[string]any{
			"id":            e.ID.String(),
			"actor_user_id": actorUserID,
			"actor_email":   e.ActorEmail,
			"action":        e.Action,
			"resource_type": e.ResourceType,
			"resource_id":   e.ResourceID,
			"summary":       e.Summary,
			"metadata":      meta,
			"created_at":    e.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": out},
	})
}
