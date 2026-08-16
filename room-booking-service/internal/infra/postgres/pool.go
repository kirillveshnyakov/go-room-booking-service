package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(
	ctx context.Context,
	dsn string,
) (*pgxpool.Pool, error) {
	pgxcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgx config: %w", err)
	}

	pgxcfg.MaxConns = 10
	pgxcfg.MinConns = 0
	pgxcfg.MinIdleConns = 2
	pgxcfg.MaxConnLifetime = 30 * time.Minute
	pgxcfg.MaxConnLifetimeJitter = 5 * time.Minute
	pgxcfg.MaxConnIdleTime = 5 * time.Minute
	pgxcfg.HealthCheckPeriod = 1 * time.Minute
	pgxcfg.PingTimeout = 2 * time.Second

	pgxcfg.ConnConfig.RuntimeParams["application_name"] = "room-booking-service"
	pgxcfg.ConnConfig.RuntimeParams["timezone"] = "UTC"

	pool, err := pgxpool.NewWithConfig(ctx, pgxcfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}
