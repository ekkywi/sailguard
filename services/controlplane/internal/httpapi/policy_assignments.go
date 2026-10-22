package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/policy"
	"github.com/google/uuid"
)

type addPolicyAssignmentRequest struct {
	Scope    string `json:"scope"`
	GroupID  string `json:"group_id"`
	DeviceID string `json:"device_id"`
}

func assignmentJSON(a policy.PolicyAssignment) map[string]any {
	out := map[string]any{
		"id":         a.ID.String(),
		"policy_id":  a.PolicyID.String(),
		"scope":      a.Scope,
		"created_at": a.CreatedAt,
	}
	if a.GroupID != nil {
		out["group_id"] = a.GroupID.String()
	} else {
		out["group_id"] = nil
	}
	if a.DeviceID != nil {
		out["device_id"] = a.DeviceID.String()
	} else {
		out["device_id"] = nil
	}
	return out
}

func (a *API) handleListPolicyAssignments(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid policy id",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	if _, err := store.FindPolicyByID(r.Context(), id); errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "policy not found",
			},
		})
		return
	} else if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load policy",
			},
		})
		return
	}

	items, err := store.ListAssignments(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list assignments",
			},
		})
		return
	}

	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, assignmentJSON(item))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": out},
	})
}

func (a *API) handleAddPolicyAssignment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid policy id",
			},
		})
		return
	}

	var req addPolicyAssignmentRequest
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
	asg, err := store.AddAssignment(r.Context(), id, policy.AddAssignmentInput{
		Scope:    strings.TrimSpace(req.Scope),
		GroupID:  strings.TrimSpace(req.GroupID),
		DeviceID: strings.TrimSpace(req.DeviceID),
	})
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "policy not found",
			},
		})
		return
	}
	if errors.Is(err, policy.ErrInvalid) {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid assignment fields (scope/group_id/device_id)",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to add assignment",
			},
		})
		return
	}

	meta := map[string]any{
		"policy_id": asg.PolicyID.String(),
		"scope":     asg.Scope,
	}
	if asg.GroupID != nil {
		meta["group_id"] = asg.GroupID.String()
	}
	if asg.DeviceID != nil {
		meta["device_id"] = asg.DeviceID.String()
	}
	a.recordAudit(r, "policy.assignment.create", "policy_assignment", asg.ID.String(),
		"Added policy assignment",
		meta,
	)

	writeJSON(w, http.StatusCreated, envelope{
		OK:   true,
		Data: assignmentJSON(*asg),
	})
}

func (a *API) handleRemovePolicyAssignment(w http.ResponseWriter, r *http.Request) {
	policyID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid policy id",
			},
		})
		return
	}
	assignmentID, err := uuid.Parse(r.PathValue("assignmentId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid assignment id",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	err = store.RemoveAssignment(r.Context(), policyID, assignmentID)
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "assignment not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to remove assignment",
			},
		})
		return
	}

	a.recordAudit(r, "policy.assignment.delete", "policy_assignment", assignmentID.String(),
		"Removed policy assignment",
		map[string]any{
			"policy_id":     policyID.String(),
			"assignment_id": assignmentID.String(),
		},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"policy_id":     policyID.String(),
			"assignment_id": assignmentID.String(),
		},
	})
}

func (a *API) handlePublishPolicy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid policy id",
			},
		})
		return
	}

	store := policy.NewStore(a.pool)
	p, err := store.PublishPolicy(r.Context(), id)
	if errors.Is(err, policy.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "policy not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to publish policy",
			},
		})
		return
	}

	a.recordAudit(r, "policy.publish", "policy", p.ID.String(),
		"published policy "+p.Name,
		map[string]any{
			"name": p.Name,
			"version": p.Version,
			"mode": p.Mode,
		},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: policyJSON(*p),
	})
}
