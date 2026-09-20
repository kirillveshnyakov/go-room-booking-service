package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const connectionRetryInterval = time.Second

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

	var lastErr error

	for {
		pool, poolErr := pgxpool.NewWithConfig(ctx, pgxcfg)
		if poolErr == nil {
			pingErr := pool.Ping(ctx)
			if pingErr == nil {
				return pool, nil
			}

			pool.Close()
			lastErr = fmt.Errorf("ping database: %w", pingErr)
		} else {
			lastErr = fmt.Errorf("create postgres pool: %w", poolErr)
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect to postgres: %w", errors.Join(lastErr, ctx.Err()))
		case <-time.After(connectionRetryInterval):
		}
	}
}
