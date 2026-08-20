package main

import (
	"log"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/app"
	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/config"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("can not initialize config: %v", err)
	}

	err = app.Run(cfg)
	if err != nil {
		log.Fatalf("can not start app: %v", err)
	}
}
