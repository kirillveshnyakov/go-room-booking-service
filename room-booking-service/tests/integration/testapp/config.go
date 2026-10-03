package testapp

import (
	"time"

	"github.com/kirillveshnyakov/go-room-booking-service/room-booking-service/internal/config"
)

func newTestConfig() *config.Config {
	return &config.Config{
		HTTP: config.HTTPConfig{
			Host:              "localhost",
			Port:              "8080",
			RateLimit:         10_000,
			RateLimitBurst:    10_000,
			ConcurrencyLimit:  100,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
			HTTPShutdownTime:  5 * time.Second,
		},

		PG: config.PGConfig{
			Host:     "localhost",
			Port:     "5433",
			DB:       "room_booking_test",
			User:     "meeting_rooms_user",
			Password: "12345",
		},

		Logger: config.LoggerConfig{
			Level:       "error",
			Environment: "development",
			Service:     "room_booking_service_test",
		},

		Auth: config.AuthConfig{
			PasswordHashCost:    4,
			JWTSecret:           "test-only-room-booking-jwt-secret-1234567890",
			AccessTokenIssuer:   "room-booking-service-test",
			AccessTokenAudience: "room-booking-api-test",
			AccessTokenTTL:      15 * time.Minute,
			SessionTTL:          24 * time.Hour,

			AuthRateLimit:      10_000,
			AuthRateLimitBurst: 10_000,

			RefreshCookieName:     "refresh_token",
			RefreshCookiePath:     "/",
			RefreshCookieSecure:   false,
			RefreshCookieSameSite: "lax",

			EnableDummyLogin: false,
		},

		Conference: config.ConferenceConfig{
			BaseURL: "https://meet.test.example.com",
		},
	}
}
