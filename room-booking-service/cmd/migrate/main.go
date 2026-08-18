package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/config"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/infra/postgres/migrate"
)

func main() {
	cfg, err := config.New()

	if err != nil {
		log.Fatalf("can not initialize config: %v", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	if err = migrate.Up(ctx, cfg.ConstructPostgresURL()); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}
}
