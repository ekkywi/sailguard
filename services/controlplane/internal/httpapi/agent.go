package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ekkywi/sailguard/pkgs/agentcontract"
	"github.com/ekkywi/sailguard/services/controlplane/internal/domain/inventory"
	"github.com/ekkywi/sailguard/services/controlplane/internal/queue"
)

func (a *API) handleAgentEnroll(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, a.cfg.EnrollMaxBody)

	var req agentcontract.EnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusBadRequest, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "validation_error",
					"message": "request body too large",
				},
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid JSON body",
			},
		})
		return
	}

	token := strings.TrimSpace(req.EnrollmentToken)
	hostname := strings.TrimSpace(req.Hostname)
	if token == "" || hostname == "" {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "enrollment_token and hostname are required",
			},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	device, deviceToken, err := store.EnrollDevice(
		r.Context(),
		token,
		hostname,
		req.OSFamily,
		req.OSVersion,
		req.AgentVersion,
		req.MachineGUID,
	)

	if err != nil {
		switch {
		case errors.Is(err, inventory.ErrEnrollmentInvalid):
			writeJSON(w, http.StatusUnauthorized, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "enrollment_invalid",
					"message": "enrollment token is invalid or expired",
				},
			})
		case errors.Is(err, inventory.ErrEnrollmentExhausted):
			writeJSON(w, http.StatusConflict, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "enrollment_exhausted",
					"message": "enrollment token has no uses remaining",
				},
			})
		case errors.Is(err, inventory.ErrDeviceExists):
			writeJSON(w, http.StatusConflict, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "device_exists",
					"message": "device already enrolled",
				},
			})
		case errors.Is(err, inventory.ErrDeviceDisabled):
			writeJSON(w, http.StatusForbidden, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "device_disabled",
					"message": "device is disabled",
				},
			})
		default:
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "internal_error",
					"message": "enrollment failed",
				},
			})
		}
		return
	}

	writeJSON(w, http.StatusCreated, envelope{
		OK: true,
		Data: map[string]any{
			"device_id":        device.ID.String(),
			"device_token":     deviceToken,
			"token_expires_at": nil,
			"policy_poll_sec":  60,
			"event_flush_sec":  30,
			"heartbeat_sec":    60,
		},
	})
}

func (a *API) handleAgentWhoami(w http.ResponseWriter, r *http.Request) {
	deviceID, ok := deviceIDFromCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "missing device auth context",
			},
		})
		return
	}
	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"device_id": deviceID.String(),
		},
	})
}

func (a *API) handleAgentEvents(w http.ResponseWriter, r *http.Request) {
	deviceID, ok := deviceIDFromCtx(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "unauthorized",
				"message": "missing device auth context",
			},
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, a.cfg.EventsMaxBody)

	var req agentcontract.EventBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeJSON(w, http.StatusBadRequest, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "validation_error",
					"message": "request body too large",
				},
			})
			return
		}
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "invalid JSON body",
			},
		})
		return
	}

	if req.SchemaVersion == 0 {
		req.SchemaVersion = 1
	}
	if len(req.Events) == 0 {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "events must not be empty",
			},
		})
		return
	}

	if len(req.Events) > a.cfg.EventsMaxBatch {
		writeJSON(w, http.StatusBadRequest, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "validation_error",
				"message": "too many events in batch",
			},
		})
		return
	}

	if strings.TrimSpace(req.DeviceID) != "" && req.DeviceID != deviceID.String() {
		writeJSON(w, http.StatusForbidden, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "forbidden",
				"message": "device_id does not match authenticated device",
			},
		})
		return
	}

	store := inventory.NewStore(a.pool)
	q := queue.New(a.rdb)
	accepted := 0
	rejected := make([]map[string]any, 0)
	sawHeartbeat := false
	var hbHostname, hbOSVersion, hbAgentVersion string

	for _, ev := range req.Events {
		clientID := strings.TrimSpace(ev.ClientEventID)
		if clientID == "" {
			rejected = append(rejected, map[string]any{
				"client_event_id": clientID,
				"code":            "validation_error",
				"message":         "client_event_id is required",
			})
			continue
		}

		switch ev.EventType {
		case agentcontract.EventHeartbeat:
			sawHeartbeat = true
			if ev.Hostname != "" {
				hbHostname = ev.Hostname
			}
			if ev.OSVersion != "" {
				hbOSVersion = ev.OSVersion
			}
			if ev.AgentVersion != "" {
				hbAgentVersion = ev.AgentVersion
			}
			accepted++
		case agentcontract.EventViolationDetected, agentcontract.EventViolationBlocked:
			if ev.Process == nil || strings.TrimSpace(ev.Process.Name) == "" {
				rejected = append(rejected, map[string]any{
					"client_event_id": clientID,
					"code":            "validation_error",
					"message":         "violation_detected requires process.name",
				})
				continue
			}

			occurredAt := time.Now().UTC()
			if ts := strings.TrimSpace(ev.OccurredAt); ts != "" {
				if t, err := time.Parse(time.RFC3339, ts); err == nil {
					occurredAt = t.UTC()
				}
			}

			msg := queue.ViolationMessage{
				DeviceID:           deviceID.String(),
				ClientEventID:      clientID,
				EventType:          ev.EventType,
				OccurredAt:         occurredAt.Format(time.RFC3339),
				DedupeKey:          ev.DedupeKey,
				ProcessPID:         ev.Process.PID,
				ProcessName:        ev.Process.Name,
				ProcessPath:        ev.Process.Path,
				ProcessHashSHA256:  ev.Process.HashSHA256,
				ProcessPublisher:   ev.Process.Publisher,
				ProcessProductName: ev.Process.ProductName,
				ProcessUserName:    ev.Process.UserName,
			}
			if ev.Match != nil {
				msg.MatchRuleID = ev.Match.RuleID
				msg.MatchAction = ev.Match.Action
				msg.MatchRuleType = ev.Match.RuleType
				msg.MatchValue = ev.Match.Value
				msg.MatchCategoryCode = ev.Match.CategoryCode
			}
			if ev.Policy != nil {
				msg.PolicyVersion = ev.Policy.Version
				msg.PolicyMode = ev.Policy.Mode
			}

			values, err := queue.EncodeViolationValues(msg)
			if err != nil {
				rejected = append(rejected, map[string]any{
					"client_event_id": clientID,
					"code":            "enqueue_error",
					"message":         "failed to encode violation for stream",
				})
				continue
			}
			if _, err := q.Produce(r.Context(), queue.StreamEvents, values); err != nil {
				rejected = append(rejected, map[string]any{
					"client_event_id": clientID,
					"code":            "enqueue_error",
					"message":         "failed to enqueue violation",
				})
				continue
			}
			accepted++
		default:
			rejected = append(rejected, map[string]any{
				"client_event_id": clientID,
				"code":            "unsupported_event",
				"message":         "event_type not accepted yet: " + ev.EventType,
			})
		}
	}

	if sawHeartbeat {
		if err := store.ApplyHeartbeat(r.Context(), deviceID, hbHostname, hbOSVersion, hbAgentVersion); err != nil {
			writeJSON(w, http.StatusInternalServerError, envelope{
				OK: false,
				Error: map[string]any{
					"code":    "internal_error",
					"message": "failed to update last_seen",
				},
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"accepted": accepted,
			"rejected": rejected,
		},
	})
}
