package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/config"
)

func NewRouter(cfg config.Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", handleHealth)
	mux.HandleFunc("GET /v1/ready", handleReady(cfg))
	mux.HandleFunc("GET /v1/system/info", handleSystemInfo(cfg))
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

func handleReady(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		// Full DB/Redis probes arrive with migrations milestone.
		writeJSON(w, http.StatusOK, envelope{
			OK: true,
			Data: map[string]any{
				"status":    "ok",
				"db_url_set": cfg.DBURL != "",
				"redis_set":  cfg.RedisURL != "",
				"note":       "connectivity probes not implemented yet",
			},
		})
	}
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
