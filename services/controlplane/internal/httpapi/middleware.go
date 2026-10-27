package httpapi

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"

	"github.com/ekkywi/sailguard/services/controlplane/internal/auth"
	"github.com/ekkywi/sailguard/services/controlplane/internal/authsession"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/identity"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/ekkywi/sailguard/services/controlplane/internal/ratelimit"
	"github.com/google/uuid"
)

type ctxKey string

const (
	ctxUserID      ctxKey = "userID"
	ctxSessionID   ctxKey = "sessionID"
	ctxPermissions ctxKey = "permissions"
	ctxDeviceID    ctxKey = "deviceID"
)

func (a *API) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r)
		if !ok || raw == "" {
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "unauthorized",
					"message": "missing bearer token",
				},
			})
			return
		}

		claims, err := auth.ParseAccessToken(a.cfg.JWTSecret, raw)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "unauthorized",
					"message": "invalid or expired token",
				},
			})
			return
		}

		jti := claims.ID
		if jti == "" {
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "unauthorized",
					"message": "invalid or expired token",
				},
			})
			return
		}

		okSession, err := authsession.New(a.rdb).Exists(r.Context(), jti)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "internal_error",
					"message": "failed to check session",
				},
			})
			return
		}
		if !okSession {
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "unauthorized",
					"message": "session revoked or expired",
				},
			})
			return
		}

		store := identity.NewStore(a.pool)
		perms, err := store.ListPermissionCodes(r.Context(), claims.UserID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "internal_error",
					"message": "failed to load permissions",
				},
			})
			return
		}

		ctx := context.WithValue(r.Context(), ctxUserID, claims.UserID)
		ctx = context.WithValue(ctx, ctxSessionID, jti)
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
					"code":    "unauthorized",
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
					"code":    "unauthorized",
					"message": "invalid device token",
				},
			})
			return
		}
		if errors.Is(err, inventory.ErrDeviceDisabled) {
			writeJSON(w, http.StatusForbidden, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "device_disabled",
					"message": "device is disabled",
				},
			})
			return
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "internal_error",
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

func sessionIDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ctxSessionID).(string)
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

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func (a *API) rateLimit(
	keyFn func(*http.Request) (string, bool),
	limit int,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, ok := keyFn(r)
		if !ok || key == "" {
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "unauthorized",
					"message": "missing rate limit key",
				},
			})
			return
		}

		allowed, err := ratelimit.New(a.rdb).Allow(r.Context(), key, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "internal_error",
					"message": "rate limit check failed",
				},
			})
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", "60")
			writeJSON(w, http.StatusTooManyRequests, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "rate_limited",
					"message": "too many requests, try again later",
				},
			})
			return
		}
		next(w, r)
	}
}

func enrollRateKey(r *http.Request) (string, bool) {
	ip := clientIP(r)
	if ip == "" {
		return "", false
	}
	return "rl:enroll:ip:" + ip, true
}

func eventsRateKey(r *http.Request) (string, bool) {
	id, ok := deviceIDFromCtx(r.Context())
	if !ok {
		return "", false
	}
	return "rl:events:dev:" + id.String(), true
}
