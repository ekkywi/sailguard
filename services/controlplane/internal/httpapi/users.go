package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/identity"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type createUserRequest struct {
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Password string   `json:"password"`
	Roles    []string `json:"roles"`
}

type patchUserRequest struct {
	Name     *string   `json:"name"`
	IsActive *bool     `json:"is_active"`
	Roles    *[]string `json:"roles"`
}

type resetPasswordRequest struct {
	Password string `json:"password"`
}

func roleCodesContain(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}

func userJSON(u identity.User, roles []string) map[string]any {
	if roles == nil {
		roles = []string{}
	}
	return map[string]any{
		"id":            u.ID.String(),
		"email":         u.Email,
		"name":          u.Name,
		"auth_provider": u.AuthProvider,
		"is_active":     u.IsActive,
		"roles":         roles,
		"last_login_at": u.LastLoginAt,
		"created_at":    u.CreatedAt,
		"updated_at":    u.UpdatedAt,
	}
}

func (a *API) handleListRoles(w http.ResponseWriter, r *http.Request) {
	store := identity.NewStore(a.pool)
	roles, err := store.ListRoles(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list roles",
			},
		})
		return
	}
	items := make([]map[string]any, 0, len(roles))
	for _, role := range roles {
		items = append(items, map[string]any{
			"code":        role.Code,
			"name":        role.Name,
			"description": role.Description,
		})
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": items},
	})
}

func (a *API) handleListPermissions(w http.ResponseWriter, r *http.Request) {
	store := identity.NewStore(a.pool)
	perms, err := store.ListPermissions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list permissions",
			},
		})
		return
	}
	items := make([]map[string]any, 0, len(perms))
	for _, p := range perms {
		items = append(items, map[string]any{
			"code":        p.Code,
			"description": p.Description,
		})
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": items},
	})
}

func (a *API) handleListUsers(w http.ResponseWriter, r *http.Request) {
	store := identity.NewStore(a.pool)
	users, err := store.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list users",
			},
		})
		return
	}

	items := make([]map[string]any, 0, len(users))
	for _, u := range users {
		roles, err := store.ListRoleCodes(r.Context(), u.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "internal_error",
					"message": "failed to load user roles",
				},
			})
			return
		}
		items = append(items, userJSON(u, roles))
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": items},
	})
}

func (a *API) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "invalid_error",
				"message": "invalid user id",
			},
		})
		return
	}

	store := identity.NewStore(a.pool)
	user, err := store.FindByID(r.Context(), id)
	if errors.Is(err, identity.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "user not found",
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

	roles, err := store.ListRoleCodes(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load user roles",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: userJSON(*user, roles),
	})
}

func (a *API) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "invalid_error",
				"message": "invalid JSON body",
			},
		})
		return
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	req.Password = strings.TrimSpace(req.Password)
	if req.Email == "" || req.Name == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "invalid_error",
				"message": "email, name, and password are required",
			},
		})
		return
	}
	if len(req.Password) < 10 {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "password must be at least 10 characters",
			},
		})
		return
	}
	if len(req.Roles) == 0 {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "at least one role is required",
			},
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
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

	store := identity.NewStore(a.pool)
	user, err := store.CreateUser(r.Context(), req.Email, req.Name, string(hash), req.Roles)
	if errors.Is(err, identity.ErrConflict) {
		writeJSON(w, http.StatusConflict, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "conflict",
				"message": "user already exsits",
			},
		})
		return
	}
	if errors.Is(err, identity.ErrInvalidRole) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "one or more roles are invalid",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to create user",
			},
		})
		return
	}

	roles, err := store.ListRoleCodes(r.Context(), user.ID)
	if err != nil {
		roles = req.Roles
	}

	a.recordAudit(r, "user.create", "user", user.ID.String(),
		"Created user "+user.Email,
		map[string]any{
			"email":  user.Email,
			"name":   user.Name,
			"roles":  roles,
			"active": user.IsActive,
		},
	)

	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: userJSON(*user, roles),
	})
}

func (a *API) handlePatchUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid user id",
			},
		})
		return
	}

	var req patchUserRequest
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
	if req.Name == nil && req.IsActive == nil && req.Roles == nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "name, is_active, or roles is required",
			},
		})
		return
	}

	actorID, ok := userIDFromCtx(r.Context())
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
	if id == actorID && req.IsActive != nil && !*req.IsActive {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "cannot deactivate yourself",
			},
		})
		return
	}

	store := identity.NewStore(a.pool)
	user, err := store.FindByID(r.Context(), id)
	if errors.Is(err, identity.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "user not found",
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

	currentRoles, err := store.ListRoleCodes(r.Context(), user.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load user roles",
			},
		})
		return
	}

	wasSuper := roleCodesContain(currentRoles, "super_admin")
	willBeActive := user.IsActive
	if req.IsActive != nil {
		willBeActive = *req.IsActive
	}
	willBeSuper := wasSuper
	if req.Roles != nil {
		willBeSuper = roleCodesContain(*req.Roles, "super_admin")
	}

	if wasSuper && user.IsActive && !(willBeActive && willBeSuper) {
		n, err := store.CountActiveSuperAdmins(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "internal_error",
					"message": "failed to check super_admin count",
				},
			})
			return
		}
		if n <= 1 {
			writeJSON(w, http.StatusBadRequest, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "validation_error",
					"message": "cannot remove the last active super_admin",
				},
			})
			return
		}
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		req.Name = &trimmed
	}

	updated, err := store.UpdateUser(r.Context(), id, req.Name, req.IsActive, req.Roles)
	if errors.Is(err, identity.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "user not found",
			},
		})
		return
	}
	if errors.Is(err, identity.ErrInvalidRole) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "one or more roles are invalid",
			},
		})
		return
	}
	if err != nil {
		msg := "failed to update user"
		if err.Error() == "name cannot be empty" {
			writeJSON(w, http.StatusBadRequest, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "validation_error",
					"message": err.Error(),
				},
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": msg,
			},
		})
		return
	}

	roles, err := store.ListRoleCodes(r.Context(), updated.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load user roles",
			},
		})
		return
	}

	a.recordAudit(r, "user.update", "user", updated.ID.String(),
		"Updated user "+updated.Email,
		map[string]any{
			"email":  updated.Email,
			"name":   updated.Name,
			"roles":  roles,
			"active": updated.IsActive,
		},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: userJSON(*updated, roles),
	})
}

func (a *API) handleResetUserPassword(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid user id",
			},
		})
		return
	}

	var req resetPasswordRequest
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
	req.Password = strings.TrimSpace(req.Password)
	if len(req.Password) < 10 {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "password must be at least 10 characters",
			},
		})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
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

	store := identity.NewStore(a.pool)
	err = store.SetPasswordHash(r.Context(), id, string(hash))
	if errors.Is(err, identity.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "user not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to reset password",
			},
		})
		return
	}

	a.recordAudit(r, "user.reset_password", "user", id.String(),
		"Reset user password",
		map[string]any{
			"user_id": id.String(),
		},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"id":      id.String(),
			"message": "password updated",
		},
	})
}
