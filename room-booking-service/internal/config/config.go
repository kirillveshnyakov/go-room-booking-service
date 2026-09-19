package config

import (
	"net"
	"net/url"
	"time"

	"github.com/caarlos0/env/v10"
)

type (
	HTTPConfig struct {
		Host              string        `env:"HTTP_HOST" envDefault:"localhost"`
		Port              string        `env:"HTTP_PORT" envDefault:"8080"`
		RateLimit         float64       `env:"HTTP_RATE_LIMIT" envDefault:"100"`
		RateLimitBurst    int           `env:"HTTP_RATE_LIMIT_BURST" envDefault:"100"`
		ConcurrencyLimit  int           `env:"HTTP_CONCURRENCY_LIMIT" envDefault:"10"`
		ReadHeaderTimeout time.Duration `env:"HTTP_READ_HEADER_TIMEOUT" envDefault:"5s"`
		ReadTimeout       time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"15s"`
		WriteTimeout      time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"30s"`
		IdleTimeout       time.Duration `env:"HTTP_IDLE_TIMEOUT" envDefault:"60s"`
		HTTPShutdownTime  time.Duration `env:"HTTP_SHUTDOWN_TIME" envDefault:"30s"`
	}
	PGConfig struct {
		Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
		Port     string `env:"POSTGRES_PORT" envDefault:"5432"`
		DB       string `env:"POSTGRES_DB" envDefault:"meeting_rooms"`
		User     string `env:"POSTGRES_USER" envDefault:"meeting_rooms_user"`
		Password string `env:"POSTGRES_PASSWORD" envDefault:"12345"`
	}

	LoggerConfig struct {
		Level       string `env:"LOGGER_LEVEL" envDefault:"info"`
		Environment string `env:"LOGGER_ENVIRONMENT" envDefault:"development"`
		Service     string `env:"LOGGER_SERVICE" envDefault:"room_booking_service"`
	}

	AuthConfig struct {
		PasswordHashCost    int           `env:"PASSWORD_HASH_COST" envDefault:"12"`
		JWTSecret           string        `env:"JWT_SECRET,required"`
		AccessTokenIssuer   string        `env:"ACCESS_TOKEN_ISSUER" envDefault:"room-booking-service"`
		AccessTokenAudience string        `env:"ACCESS_TOKEN_AUDIENCE" envDefault:"room-booking-api"`
		AccessTokenTTL      time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
		SessionTTL          time.Duration `env:"SESSION_TTL" envDefault:"720h"`
		RefreshCookieSecure bool          `env:"REFRESH_COOKIE_SECURE" envDefault:"false"`
	}

	ConferenceConfig struct {
		BaseURL string `env:"CONFERENCE_BASE_URL" envDefault:"https://meet.example.com"`
	}

	Config struct {
		HTTP       HTTPConfig
		PG         PGConfig
		Logger     LoggerConfig
		Auth       AuthConfig
		Conference ConferenceConfig
	}
)

func (c *HTTPConfig) HTTPAddress() string {
	return net.JoinHostPort(c.Host, c.Port)
}

func (c *Config) ConstructPostgresURL() string {
	postgresURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.PG.User, c.PG.Password),
		Host:   net.JoinHostPort(c.PG.Host, c.PG.Port),
		Path:   c.PG.DB,
	}
	query := postgresURL.Query()
	query.Set("sslmode", "disable")
	postgresURL.RawQuery = query.Encode()

	return postgresURL.String()
}
func New() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	return &cfg, err
}
