package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/alerting"
	"github.com/google/uuid"
)

func alertChannelJSON(ch alerting.Channel) map[string]any {
	url := ch.WebhookURL()
	return map[string]any{
		"id":         ch.ID.String(),
		"kind":       ch.Kind,
		"name":       ch.Name,
		"enabled":    ch.Enabled,
		"url":        url,
		"created_at": ch.CreatedAt,
		"updated_at": ch.UpdatedAt,
	}
}

func (a *API) handleListAlertChannels(w http.ResponseWriter, r *http.Request) {
	items, err := alerting.NewStore(a.pool).ListChannels(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to list alert channels",
			},
		})
		return
	}
	out := make([]map[string]any, 0, len(items))
	for _, ch := range items {
		out = append(out, alertChannelJSON(ch))
	}
	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"items": out},
	})
}

type patchAlertChannelRequest struct {
	Name    *string `json:"name"`
	Enabled *bool   `json:"enabled"`
	URL     *string `json:"url"`
}

func (a *API) handlePatchAlertChannel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid id",
			},
		})
		return
	}

	var req patchAlertChannelRequest
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
	if req.Name == nil && req.Enabled == nil && req.URL == nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "no fields to update",
			},
		})
		return
	}
	if req.URL != nil {
		u := strings.TrimSpace(*req.URL)
		if u != "" && !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			writeJSON(w, http.StatusBadRequest, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "validation_error",
					"message": "url must start with http:// or https://",
				},
			})
			return
		}
		req.URL = &u
	}

	ch, err := alerting.NewStore(a.pool).UpdateChannel(r.Context(), id, alerting.UpdateChannelInput{
		Name:    req.Name,
		Enabled: req.Enabled,
		URL:     req.URL,
	})
	if errors.Is(err, alerting.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "alert channel not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to update alert channel",
			},
		})
		return
	}

	a.recordAudit(r, "alert.channel.update", "alert_channel", ch.ID.String(),
		"Updated alert channel",
		map[string]any{"enabled": ch.Enabled, "kind": ch.Kind},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: alertChannelJSON(*ch),
	})
}

func (a *API) handleTestAlertChannel(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid id",
			},
		})
		return
	}

	ch, err := alerting.NewStore(a.pool).GetChannel(r.Context(), id)
	if errors.Is(err, alerting.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_found",
				"message": "alert channel not found",
			},
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "internal_error",
				"message": "failed to load alert channel",
			},
		})
		return
	}

	url := strings.TrimSpace(ch.WebhookURL())
	if url == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "webhook url is empty",
			},
		})
		return
	}

	payload := map[string]any{
		"type":       "sailguard.test",
		"message":    "SailGuard webhook test",
		"channel_id": ch.ID.String(),
		"sent_at":    time.Now().UTC().Format(time.RFC3339),
	}
	if err := alerting.DeliverWebhook(r.Context(), url, payload); err != nil {
		writeJSON(w, http.StatusBadGateway, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "delivery_failed",
				"message": err.Error(),
			},
		})
		return
	}

	a.recordAudit(r, "alert.channel.test", "alert_channel", ch.ID.String(),
		"Sent test webhook",
		map[string]any{"kind": ch.Kind},
	)

	writeJSON(w, http.StatusOK, envelope{
		OK:   true,
		Data: map[string]any{"sent": true},
	})
}
