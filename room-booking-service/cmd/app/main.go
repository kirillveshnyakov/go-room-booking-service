package main

import (
	"log"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/app"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/config"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()

	if err != nil {
		log.Fatalf("can not initialize logger: %s", err)
	}

	cfg, err := config.New()

	if err != nil {
		log.Fatalf("can not initialize config: %s", err)
	}

	app.Run(logger, cfg)
}
