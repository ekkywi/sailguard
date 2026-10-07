package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/google/uuid"
)

func enrollmentTokenJSON(t inventory.EnrollmentToken) map[string]any {
	var createdBy any
	if t.CreatedBy != nil {
		createdBy = t.CreatedBy.String()
	}
	return map[string]any{
		"id":         t.ID.String(),
		"label":      t.Label,
		"max_uses":   t.MaxUses,
		"use_count":  t.UseCount,
		"expires_at": t.ExpiresAt,
		"revoked_at": t.RevokedAt,
		"created_by": createdBy,
		"created_at": t.CreatedAt,
	}
}

type createEnrollmentTokenRequest struct {
	Label     string `json:"label"`
	MaxUses   int    `json:"max_uses"`
	ExpiresIn *int   `json:"expires_in_hours"`
}

func (a *API) handleListEnrollmentTokens(w http.ResponseWriter, r *http.Request) {
	store := inventory.NewStore(a.pool)
	tokens, err := store.ListEnrollmentTokens(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{"code": "internal_error", "message": "failed to list enrollment tokens"},
		})
		return
	}

	items := make([]map[string]any, 0, len(tokens))
	for _, t := range tokens {
		items = append(items, enrollmentTokenJSON(t))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{"items": items},
	})
}

func (a *API) handleCreateEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	var req createEnrollmentTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{"code": "validation_error", "message": "invalid JSON body"},
		})
		return
	}

	label := strings.TrimSpace(req.Label)
	maxUses := req.MaxUses
	if maxUses <= 0 {
		maxUses = 1
	}

	var expiresAt *time.Time
	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Hour)
		expiresAt = &t
	}

	secret, err := inventory.GenerateEnrollmentSecret()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{"code": "internal_error", "message": "failed to generate token"},
		})
		return
	}
	hash := inventory.HashToken(secret)

	var createdBy *uuid.UUID
	if uid, ok := userIDFromCtx(r.Context()); ok {
		createdBy = &uid
	}

	store := inventory.NewStore(a.pool)
	tok, err := store.CreateEnrollmentToken(r.Context(), label, hash, maxUses, expiresAt, createdBy)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{"code": "internal_error", "message": "failed to create enrollment token"},
		})
		return
	}

	data := enrollmentTokenJSON(*tok)
	data["token"] = secret

	writeJSON(w, http.StatusCreated, envelope{OK: true, Data: data})
}

func (a *API) handleRevokeEnrollmentToken(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{"code": "validation_error", "message": "invalid token id"},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	tok, err := store.RevokeEnrollmentToken(r.Context(), id)
	if errors.Is(err, inventory.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{"code": "not_found", "message": "enrollment token not found or already revoked"},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{"code": "internal_error", "message": "failed to revoke enrollment token"},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: enrollmentTokenJSON(*tok),
	})
}