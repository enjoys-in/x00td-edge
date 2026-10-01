package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func NewRouter(logger *slog.Logger, pool *pgxpool.Pool, rdb *redis.Client, cacheTTL time.Duration) http.Handler {
	h := &handlers{logger: logger, pool: pool, rdb: rdb, cacheTTL: cacheTTL}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /db/time", h.dbTime)

	return logRequests(logger, mux)
}

type handlers struct {
	logger   *slog.Logger
	pool     *pgxpool.Pool
	rdb      *redis.Client
	cacheTTL time.Duration
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	status := map[string]string{"status": "ok", "db": "ok", "redis": "ok"}
	code := http.StatusOK

	if err := h.pool.Ping(ctx); err != nil {
		status["db"] = "down"
		status["status"] = "degraded"
		code = http.StatusServiceUnavailable
	}
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		status["redis"] = "down"
		status["status"] = "degraded"
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, status)
}

// dbTime demonstrates the cache-miss (Postgres round-trip) vs cache-hit (Redis)
// path — the core cost the benchmarking plan measures.
func (h *handlers) dbTime(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	const key = "db:time"
	if cached, err := h.rdb.Get(ctx, key).Result(); err == nil {
		writeJSON(w, http.StatusOK, map[string]any{"time": cached, "source": "cache"})
		return
	} else if !errors.Is(err, redis.Nil) {
		h.logger.Warn("redis get failed", "err", err)
	}

	var ts time.Time
	if err := h.pool.QueryRow(ctx, "SELECT now()").Scan(&ts); err != nil {
		h.logger.Error("db query failed", "err", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "db query failed"})
		return
	}
	val := ts.Format(time.RFC3339Nano)

	if err := h.rdb.Set(ctx, key, val, h.cacheTTL).Err(); err != nil {
		h.logger.Warn("redis set failed", "err", err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"time": val, "source": "db"})
}

func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "dur", time.Since(start).String())
	})
}
