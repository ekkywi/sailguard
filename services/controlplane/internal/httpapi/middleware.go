package httpapi

import (
	"context"
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/auth"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/identity"
	"github.com/google/uuid"
)

type ctxKey string

const (
	ctxUserID      ctxKey = "userID"
	ctxPermissions ctxKey = "permissions"
)

func (a *API) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r)
		if !ok || raw == "" {
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK:    false,
				Error: map[string]any{"code": "unauthorized", "message": "missing bearer token"},
			})
			return
		}
		claims, err := auth.ParseAccessToken(a.cfg.JWTSecret, raw)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK:    false,
				Error: map[string]any{"code": "unauthorized", "message": "invalid or expired token"},
			})
			return
		}

		store := identity.NewStore(a.pool)
		perms, err := store.ListPermissionCodes(r.Context(), claims.UserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK:    false,
				Error: map[string]any{"code": "internal_error", "message": "failed to load permissions"},
			})
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxPermissions, perms)
		next(w, r.WithContext(ctx))
	}
}

func (a *API) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	return a.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		perms, _ := r.Context().Value(ctxPermissions).([]string)
		for _, p := range perms {
			if p == code {
				next(w, r)
				return
			}
		}
		writeJSON(w, http.StatusForbidden, envelope{
			OK:    false,
			Error: map[string]any{"code": "forbidden", "message": "missing permission: " + code},
		})
	})
}

func userIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxUserID).(uuid.UUID)
	return id, ok
}

func permissionsFromCtx(ctx context.Context) ([]string, bool) {
	perms, ok := ctx.Value(ctxPermissions).([]string)
	return perms, ok
}
