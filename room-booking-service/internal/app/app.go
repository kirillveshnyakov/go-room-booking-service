package app

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/config"
	loggerpkg "github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/logger"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres"
)

func Run(cfg *config.Config) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	logger, err := loggerpkg.New(loggerpkg.Config{
		Level:       cfg.Logger.Level,
		Environment: cfg.Logger.Environment,
		Service:     cfg.Logger.Service,
	})
	if err != nil {
		return fmt.Errorf("initialize logger: %w", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	poolCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(poolCtx, cfg.ConstructPostgresURL())
	if err != nil {
		return fmt.Errorf("initialize postgres pool: %w", err)
	}
	defer pool.Close()

	return nil
}
