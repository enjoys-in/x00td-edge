package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New builds a persistent pooled connection to the remote Postgres. A warm pool
// avoids paying TCP+TLS handshake per query, which matters over the network link.
func New(ctx context.Context, url string, maxConns, minConns int32) (*pgxpool.Pool, error) {
	pcfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pcfg.MaxConns = maxConns
	pcfg.MinConns = minConns
	pcfg.MaxConnIdleTime = 5 * time.Minute
	pcfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
