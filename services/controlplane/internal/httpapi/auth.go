package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekkywi/sailguard/services/controlplane/internal/auth"
	"github.com/ekkywi/sailguard/services/controlplane/internal/authsession"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/audit"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/identity"
	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) < len(prefix) || h[:len(prefix)] != prefix {
		return "", false
	}
	return h[len(prefix):], true
}

func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK:    false,
			Error: map[string]any{"code": "unauthorized", "message": "missing auth context"},
		})
		return
	}

	store := identity.NewStore(a.pool)
	user, err := store.FindByID(r.Context(), userID)
	if errors.Is(err, identity.ErrNotFound) || (user != nil && !user.IsActive) {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK:    false,
			Error: map[string]any{"code": "unauthorized", "message": "user not found or inactive"},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to load user"},
		})
		return
	}

	roles, err := store.ListRoleCodes(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "failed to load roles"},
		})
		return
	}
	if roles == nil {
		roles = []string{}
	}

	perms, ok := permissionsFromCtx(r.Context())
	if !ok || perms == nil {
		perms = []string{}
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"id":          user.ID.String(),
			"email":       user.Email,
			"name":        user.Name,
			"roles":       roles,
			"permissions": perms,
		},
	})
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "invalid JSON body"},
		})
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK:    false,
			Error: map[string]any{"code": "validation_error", "message": "email and password are required"},
		})
		return
	}

	store := identity.NewStore(a.pool)
	user, err := store.FindByEmail(r.Context(), req.Email)
	if errors.Is(err, identity.ErrNotFound) {
		a.writeAudit(r.Context(), audit.RecordInput{
			ActorEmail:   req.Email,
			Action:       "auth.login.fail",
			ResourceType: "user",
			Summary:      "Login failed",
			Metadata:     map[string]any{"reason": "unknown_user"},
		})
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK:    false,
			Error: map[string]any{"code": "unauthorized", "message": "invalid email or password"},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "login failed"},
		})
		return
	}

	uid := user.ID
	failLogin := func(reason string) {
		a.writeAudit(r.Context(), audit.RecordInput{
			ActorUserID:  &uid,
			ActorEmail:   user.Email,
			Action:       "auth.login.fail",
			ResourceType: "user",
			ResourceID:   uid.String(),
			Summary:      "Login failed",
			Metadata:     map[string]any{"reason": reason},
		})
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK:    false,
			Error: map[string]any{"code": "unauthorized", "message": "invalid email or password"},
		})
	}

	if !user.IsActive || user.PasswordHash == "" {
		failLogin("inactive_or_no_password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		failLogin("bad_password")
		return
	}

	ttl := a.cfg.AdminTokenTTL
	token, jti, err := auth.IssueAccessToken(a.cfg.JWTSecret, user.ID, user.Email, ttl)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "could not issue token"},
		})
		return
	}

	ua := r.Header.Get("User-Agent")
	if len(ua) > 512 {
		ua = ua[:512]
	}
	meta := authsession.SessionMeta{
		IP:        clientIP(r),
		UserAgent: ua,
	}

	if err := authsession.New(a.rdb).Create(r.Context(), user.ID, jti, ttl, meta); err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "could not create session"},
		})
		return
	}

	a.writeAudit(r.Context(), audit.RecordInput{
		ActorUserID:  &uid,
		ActorEmail:   user.Email,
		Action:       "auth.login",
		ResourceType: "user",
		ResourceID:   uid.String(),
		Summary:      "Login success",
		Metadata:     map[string]any{},
	})

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"access_token": token,
			"token_type":   "Bearer",
			"expires_in":   int(ttl.Seconds()),
			"user": map[string]any{
				"id":    user.ID.String(),
				"email": user.Email,
				"name":  user.Name,
			},
		},
	})
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK:    false,
			Error: map[string]any{"code": "unauthorized", "message": "missing auth context"},
		})
		return
	}
	jti, ok := sessionIDFromCtx(r.Context())
	if !ok || jti == "" {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK:    false,
			Error: map[string]any{"code": "unauthorized", "message": "missing session"},
		})
		return
	}

	if err := authsession.New(a.rdb).Revoke(r.Context(), userID, jti); err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK:    false,
			Error: map[string]any{"code": "internal_error", "message": "could not revoke session"},
		})
		return
	}

	email := ""
	if u, err := identity.NewStore(a.pool).FindByID(r.Context(), userID); err == nil && u != nil {
		email = u.Email
	}
	a.writeAudit(r.Context(), audit.RecordInput{
		ActorUserID:  &userID,
		ActorEmail:   email,
		Action:       "auth.logout",
		ResourceType: "user",
		ResourceID:   userID.String(),
		Summary:      "Logout",
		Metadata:     map[string]any{},
	})

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{},
	})
}

func (a *API) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "missing auth context",
			},
		})
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid JSON body",
			},
		})
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "current_password and new_password are required",
			},
		})
		return
	}
	if len(req.NewPassword) < 10 {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "password must be at least 10 characters long",
			},
		})
		return
	}
	if req.NewPassword == req.CurrentPassword {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "current_password must be different from new_password",
			},
		})
		return
	}

	store := identity.NewStore(a.pool)
	user, err := store.FindByID(r.Context(), userID)
	if errors.Is(err, identity.ErrNotFound) || user == nil || !user.IsActive {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "user not found or inactive",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load user",
			},
		})
		return
	}

	if user.AuthProvider != "local" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "password change is only available for loca accounts",
			},
		})
		return
	}
	if user.PasswordHash == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "account has no local password",
			},
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		a.writeAudit(r.Context(), audit.RecordInput{
			ActorUserID:  &userID,
			ActorEmail:   user.Email,
			Action:       "auth.password_change.fail",
			ResourceType: "user",
			ResourceID:   userID.String(),
			Summary:      "password change failed",
			Metadata: map[string]any{
				"reason": "bad_current_password",
			},
		})
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "current password is incorrect",
			},
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to hash password",
			},
		})
		return
	}

	if err := store.SetPasswordHash(r.Context(), userID, string(hash)); err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to update password",
			},
		})
		return
	}

	a.writeAudit(r.Context(), audit.RecordInput{
		ActorUserID:  &userID,
		ActorEmail:   user.Email,
		Action:       "auth.password_change",
		ResourceType: "user",
		ResourceID:   userID.String(),
		Summary:      "Password changed",
		Metadata:     map[string]any{},
	})

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{},
	})
}

func (a *API) handleListSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "missing auth context",
			},
		})
		return
	}
	currentJTI, _ := sessionIDFromCtx(r.Context())

	sessions, err := authsession.New(a.rdb).List(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list sessions",
			},
		})
		return
	}

	items := make([]map[string]any, 0, len(sessions))
	for _, s := range sessions {
		created := any(nil)
		if !s.Meta.CreatedAt.IsZero() {
			created = s.Meta.CreatedAt.UTC()
		}
		items = append(items, map[string]any{
			"id":         s.ID,
			"created_at": created,
			"ip":         s.Meta.IP,
			"user_agent": s.Meta.UserAgent,
			"current":    s.ID == currentJTI,
		})
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"items": items,
		},
	})
}

func (a *API) handleRevokeSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "missing auth context",
			},
		})
		return
	}
	target := strings.TrimSpace(r.PathValue("id"))
	if target == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "session id is required",
			},
		})
		return
	}

	store := authsession.New(a.rdb)
	sess, err := store.Get(r.Context(), target)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load session",
			},
		})
		return
	}
	if sess == nil || (sess.Meta.UserID != "" && sess.Meta.UserID != userID.String()) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "session not found",
			},
		})
		return
	}
	owned := false
	if list, lerr := store.List(r.Context(), userID); lerr == nil {
		for _, s := range list {
			if s.ID == target {
				owned = true
				break
			}
		}
	}
	if !owned {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "session not found",
			},
		})
		return
	}

	if err := store.Revoke(r.Context(), userID, target); err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to revoke session",
			},
		})
		return
	}

	email := ""
	if u, err := identity.NewStore(a.pool).FindByID(r.Context(), userID); err == nil && u != nil {
		email = u.Email
	}
	a.writeAudit(r.Context(), audit.RecordInput{
		ActorUserID:  &userID,
		ActorEmail:   email,
		Action:       "auth.session_revoke",
		ResourceType: "session",
		ResourceID:   target,
		Summary:      "Session revoked",
		Metadata:     map[string]any{},
	})

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{},
	})
}

func (a *API) handleRevokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "missing auth context",
			},
		})
		return
	}
	currentJTI, ok := sessionIDFromCtx(r.Context())
	if !ok || currentJTI == "" {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "missing session",
			},
		})
		return
	}

	if err := authsession.New(a.rdb).RevokeOthers(r.Context(), userID, currentJTI); err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to revoke sessions",
			},
		})
		return
	}

	email := ""
	if u, err := identity.NewStore(a.pool).FindByID(r.Context(), userID); err == nil && u != nil {
		email = u.Email
	}
	a.writeAudit(r.Context(), audit.RecordInput{
		ActorUserID:  &userID,
		ActorEmail:   email,
		Action:       "auth.session.revoke_others",
		ResourceType: "user",
		ResourceID:   userID.String(),
		Summary:      "Other sessions revoked",
		Metadata:     map[string]any{},
	})

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{},
	})
}
