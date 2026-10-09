package httpapi

import (
	"context"
	"net/http"
	"errors"

	"github.com/ekkywi/sailguard/services/controlplane/internal/auth"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/identity"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/google/uuid"
)

type ctxKey string

const (
	ctxUserID      ctxKey = "userID"
	ctxPermissions ctxKey = "permissions"
	ctxDeviceID    ctxKey = "deviceID"
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

func (a *API) requireDeviceAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r)
		if !ok || raw == "" {
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK: false,
				Error: map[string]any{
					"code": "unauthorized",
					"message": "missing bearer token",
				},
			})
			return
		}

		store := inventory.NewStore(a.pool)
		deviceID, err := store.FindDeviceIDByActiveTokenHash(
			r.Context(),
			inventory.HashToken(raw),
		)
		if errors.Is(err, inventory.ErrNotFound) {
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK: false,
				Error: map[string]any{
					"code": "unauthorized",
					"message": "invalid device token",
				},
			})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code": "internal_error",
					"message": "failed to authenticate device",
				},
			})
			return
		}

		ctx := context.WithValue(r.Context(), ctxDeviceID, deviceID)
		next(w, r.WithContext(ctx))
	}
}

func userIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxUserID).(uuid.UUID)
	return id, ok
}

func permissionsFromCtx(ctx context.Context) ([]string, bool) {
	perms, ok := ctx.Value(ctxPermissions).([]string)
	return perms, ok
}

func deviceIDFromCtx(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(ctxDeviceID).(uuid.UUID)
	return id, ok
}
