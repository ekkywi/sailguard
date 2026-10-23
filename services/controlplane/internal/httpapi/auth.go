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

	if err := authsession.New(a.rdb).Create(r.Context(), user.ID, jti, ttl); err != nil {
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
