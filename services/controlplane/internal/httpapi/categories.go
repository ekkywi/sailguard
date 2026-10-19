package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/policy"
	"github.com/google/uuid"
)

type createCategoryRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type patchCategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type addSignatureRequest struct {
	RuleType string `json:"rule_type"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

func categoryJSON(c policy.Category) map[string]any {
	return map[string]any{
		"code":        c.Code,
		"name":        c.Name,
		"description": c.Description,
		"created_at":  c.CreatedAt,
		"updated_at":  c.UpdatedAt,
	}
}

func categorySignatureJSON(s policy.CategorySignature) map[string]any {
	return map[string]any{
		"id":            s.ID.String(),
		"category_code": s.CategoryCode,
		"rule_type":     s.RuleType,
		"operator":      s.Operator,
		"value":         s.Value,
		"created_at":    s.CreatedAt,
	}
}

func (a *API) handleListCategories(w http.ResponseWriter, r *http.Request) {
	store := policy.NewStore(a.pool)
	items, err := store.ListCategories(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list categories",
			},
		})
		return
	}

	out := make([]map[string]any, 0, len(items))
	for _, c := range items {
		out = append(out, categoryJSON(c))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": out},
	})
}

func (a *API) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req createCategoryRequest
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

	store := policy.NewStore(a.pool)
	c, err := store.CreateCategory(r.Context(), policy.CreateCategoryInput{
		Code:        strings.TrimSpace(req.Code),
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
	})
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid category fields (code/name)",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrConflict) {
		writeJSON(w, http.StatusConflict, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "conflict",
				"message": "category code already exists",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to create category",
			},
		})
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: categoryJSON(*c),
	})
}

func (a *API) handleGetCategory(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	store := policy.NewStore(a.pool)
	c, err := store.FindCategoryByCode(r.Context(), code)
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid category code",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "category not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load category",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: categoryJSON(*c),
	})
}

func (a *API) handlePatchCategory(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	var req patchCategoryRequest
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

	store := policy.NewStore(a.pool)
	c, err := store.UpdateCategory(r.Context(), code, policy.UpdateCategoryInput{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
	})
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid category fields",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "category not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to update category",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: categoryJSON(*c),
	})
}

func (a *API) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	store := policy.NewStore(a.pool)
	err := store.DeleteCategory(r.Context(), code)
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid category code",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "category not found",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrConflict) {
		writeJSON(w, http.StatusConflict, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "conflict",
				"message": "category is referenced by policy rules; remove those rules first",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to delete category",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"deleted": true, "code": strings.TrimSpace(strings.ToLower(code))},
	})
}

func (a *API) handleListCategorySignatures(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	store := policy.NewStore(a.pool)
	items, err := store.ListSignaturesByCategory(r.Context(), code)
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid category code",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "category not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list signatures",
			},
		})
		return
	}

	out := make([]map[string]any, 0, len(items))
	for _, sig := range items {
		out = append(out, categorySignatureJSON(sig))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": out},
	})
}

func (a *API) handleAddCategorySignature(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	var req addSignatureRequest
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

	store := policy.NewStore(a.pool)
	sig, err := store.AddSignature(r.Context(), code, policy.AddSignatureInput{
		RuleType: strings.TrimSpace(req.RuleType),
		Operator: strings.TrimSpace(req.Operator),
		Value:    strings.TrimSpace(req.Value),
	})
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid signature fields",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "category not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to add signature",
			},
		})
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: categorySignatureJSON(*sig),
	})
}

func (a *API) handleDeleteCategorySignature(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	sigID, err := uuid.Parse(r.PathValue("signatureId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid signature id",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	err = store.DeleteSignature(r.Context(), code, sigID)
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid category code",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "signature not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to delete signature",
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"deleted":       true,
			"category_code": strings.TrimSpace(strings.ToLower(code)),
			"signature_id":  sigID.String(),
		},
	})
}
