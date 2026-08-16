package app

import (
	"context"
	"time"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/config"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres"

	"go.uber.org/zap"
)

func Run(logger *zap.Logger, cfg *config.Config) {
	ctx := context.Background()

	poolCtx, cancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)
	defer cancel()

	pool, err := postgres.NewPool(poolCtx, cfg.ConstructPostgresURL())
	if err != nil {
		logger.Error("failed to start application", zap.Error(err))
		return
	}
	defer pool.Close()

}
