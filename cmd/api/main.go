package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"x00td-edge/internal/cache"
	"x00td-edge/internal/config"
	"x00td-edge/internal/db"
	"x00td-edge/internal/httpapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.New(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		logger.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	rdb := cache.New(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer func() { _ = rdb.Close() }()

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           httpapi.NewRouter(logger, pool, rdb, cfg.CacheTTL),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	}
}
