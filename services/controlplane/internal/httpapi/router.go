package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/ekkywi/sailguard/services/controlplane/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type API struct {
	cfg  config.Config
	pool *pgxpool.Pool
	rdb  *redis.Client
}

func NewRouter(cfg config.Config, pool *pgxpool.Pool, rdb *redis.Client) http.Handler {
	api := &API{cfg: cfg, pool: pool, rdb: rdb}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", handleHealth)
	mux.HandleFunc("GET /v1/ready", api.handleReady)
	mux.HandleFunc("GET /v1/system/info", handleSystemInfo(cfg))

	mux.HandleFunc("GET /v1/auth/me", api.requireAuth(api.handleMe))
	mux.HandleFunc("POST /v1/auth/login", api.handleLogin)

	mux.HandleFunc("GET /v1/groups", api.requirePermission("device.read", api.handleListGroups))
	mux.HandleFunc("POST /v1/groups", api.requirePermission("device.write", api.handleCreateGroup))
	mux.HandleFunc("GET /v1/groups/{id}", api.requirePermission("device.read", api.handleGetGroup))

	mux.HandleFunc("GET /v1/devices", api.requirePermission("device.read", api.handleListDevices))
	mux.HandleFunc("GET /v1/devices/{id}", api.requirePermission("device.read", api.handleGetDevice))

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

	dbOK := a.pool.Ping(ctx) == nil
	redisOK := a.rdb.Ping(ctx).Err() == nil

	data := map[string]any{
		"db":    dbOK,
		"redis": redisOK,
	}

	if !dbOK || !redisOK {
		msg := "dependency unavailable"
		switch {
		case !dbOK && !redisOK:
			msg = "database and redis unavailable"
		case !dbOK:
			msg = "database unavailable"
		case !redisOK:
			msg = "redis unavailable"
		}
		writeJSON(w, http.StatusServiceUnavailable, envelope{
			OK: false,
			Error: map[string]any{
				"code": "not_ready",
				"message": msg,
			},
			Data: data,
		})
		return
	}

	data["status"] = "ok"
	writeJSON(w, http.StatusOK, envelope{OK: true, Data: data})
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
