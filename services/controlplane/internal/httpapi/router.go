package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	cfg  config.Config
	pool *pgxpool.Pool
}

func NewRouter(cfg config.Config, pool *pgxpool.Pool) http.Handler {
	api := &API{cfg: cfg, pool: pool}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", handleHealth)
	mux.HandleFunc("GET /v1/ready", api.handleReady)
	mux.HandleFunc("GET /v1/system/info", handleSystemInfo(cfg))

	mux.HandleFunc("GET /v1/auth/me", api.requireAuth(api.handleMe))
	mux.HandleFunc("POST /v1/auth/login", api.handleLogin)

	_ = api
	return mux
}

type envelope struct {
	OK    bool           `json:"ok"`
	Data  map[string]any `json:"data,omitempty"`
	Error map[string]any `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, envelope{OK: true, Data: map[string]any{"status": "ok"}})
}

func (a *API) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := a.pool.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, envelope{
			OK: false,
			Error: map[string]any{
				"code":    "not_ready",
				"message": "database unavailable",
			},
			Data: map[string]any{
				"db":    false,
				"redis": nil,
			},
		})
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		OK: true,
		Data: map[string]any{
			"status": "ok",
			"db":     true,
			"redis":  nil,
		},
	})
}

func handleSystemInfo(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, envelope{
			OK: true,
			Data: map[string]any{
				"name":    "sailguard-controlplane",
				"env":     cfg.Env,
				"version": "0.0.0-dev",
			},
		})
	}
}
