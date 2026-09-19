package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
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
		JWTSecret           string        `env:"JWT_SECRET" envDefault:"dev-only-room-booking-jwt-secret-change-me"`
		AccessTokenIssuer   string        `env:"ACCESS_TOKEN_ISSUER" envDefault:"room-booking-service"`
		AccessTokenAudience string        `env:"ACCESS_TOKEN_AUDIENCE" envDefault:"room-booking-api"`
		AccessTokenTTL      time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
		SessionTTL          time.Duration `env:"SESSION_TTL" envDefault:"720h"`

		AuthRateLimit      float64 `env:"HTTP_AUTH_RATE_LIMIT" envDefault:"5"`
		AuthRateLimitBurst int     `env:"HTTP_AUTH_RATE_LIMIT_BURST" envDefault:"10"`

		RefreshCookieName     string `env:"REFRESH_COOKIE_NAME" envDefault:"refresh_token"`
		RefreshCookiePath     string `env:"REFRESH_COOKIE_PATH" envDefault:"/"`
		RefreshCookieSecure   bool   `env:"REFRESH_COOKIE_SECURE" envDefault:"false"`
		RefreshCookieSameSite string `env:"REFRESH_COOKIE_SAME_SITE" envDefault:"lax"`

		EnableDummyLogin bool `env:"ENABLE_DUMMY_LOGIN" envDefault:"true"`
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

func (c *HTTPConfig) Validate() error {
	if strings.TrimSpace(c.Port) == "" {
		return fmt.Errorf("http port is required")
	}

	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("http port must be between 1 and 65535")
	}

	if c.RateLimit <= 0 {
		return fmt.Errorf("http rate limit must be positive")
	}

	if c.RateLimitBurst <= 0 {
		return fmt.Errorf("http rate limit burst must be positive")
	}

	if c.ConcurrencyLimit <= 0 {
		return fmt.Errorf("http concurrency limit must be positive")
	}

	if c.ReadHeaderTimeout <= 0 {
		return fmt.Errorf("http read header timeout must be positive")
	}

	if c.ReadTimeout <= 0 {
		return fmt.Errorf("http read timeout must be positive")
	}

	if c.WriteTimeout <= 0 {
		return fmt.Errorf("http write timeout must be positive")
	}

	if c.IdleTimeout <= 0 {
		return fmt.Errorf("http idle timeout must be positive")
	}

	if c.HTTPShutdownTime <= 0 {
		return fmt.Errorf("http shutdown time must be positive")
	}

	return nil
}

func (c *PGConfig) Validate() error {
	if strings.TrimSpace(c.Host) == "" {
		return fmt.Errorf("postgres host is required")
	}

	port, err := strconv.Atoi(c.Port)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("postgres port must be between 1 and 65535")
	}

	if strings.TrimSpace(c.DB) == "" {
		return fmt.Errorf("postgres database is required")
	}

	if strings.TrimSpace(c.User) == "" {
		return fmt.Errorf("postgres user is required")
	}

	if c.Password == "" {
		return fmt.Errorf("postgres password is required")
	}

	return nil
}

func (c *LoggerConfig) Validate() error {
	switch strings.ToLower(strings.TrimSpace(c.Level)) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("unsupported logger level: %q", c.Level)
	}

	switch strings.ToLower(strings.TrimSpace(c.Environment)) {
	case "development", "production":
	default:
		return fmt.Errorf("unsupported logger environment: %q", c.Environment)
	}

	if strings.TrimSpace(c.Service) == "" {
		return fmt.Errorf("logger service is required")
	}

	return nil
}

func (c *AuthConfig) Validate() error {
	if c.PasswordHashCost < 4 || c.PasswordHashCost > 31 {
		return fmt.Errorf("password hash cost must be between 4 and 31")
	}

	if len([]byte(c.JWTSecret)) < 32 {
		return fmt.Errorf("jwt secret must be at least 32 bytes")
	}

	if strings.TrimSpace(c.AccessTokenIssuer) == "" {
		return fmt.Errorf("access token issuer is required")
	}

	if strings.TrimSpace(c.AccessTokenAudience) == "" {
		return fmt.Errorf("access token audience is required")
	}

	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("access token ttl must be positive")
	}

	if c.SessionTTL <= 0 {
		return fmt.Errorf("session ttl must be positive")
	}

	if c.SessionTTL <= c.AccessTokenTTL {
		return fmt.Errorf("session ttl must be greater than access token ttl")
	}

	if strings.TrimSpace(c.RefreshCookieName) == "" {
		return fmt.Errorf("refresh cookie name is required")
	}

	if !strings.HasPrefix(c.RefreshCookiePath, "/") {
		return fmt.Errorf("refresh cookie path must start with /")
	}

	switch strings.ToLower(c.RefreshCookieSameSite) {
	case "lax", "strict", "none":
	default:
		return fmt.Errorf(
			"refresh cookie same site must be one of: lax, strict, none",
		)
	}

	if strings.EqualFold(c.RefreshCookieSameSite, "none") &&
		!c.RefreshCookieSecure {
		return fmt.Errorf(
			"refresh cookie secure must be true when SameSite=None",
		)
	}

	if c.AuthRateLimit <= 0 {
		return fmt.Errorf("http auth rate limit must be positive")
	}

	if c.AuthRateLimitBurst <= 0 {
		return fmt.Errorf("http auth rate limit burst must be positive")
	}

	return nil
}

func (c *ConferenceConfig) Validate() error {
	rawURL := strings.TrimSpace(c.BaseURL)
	if rawURL == "" {
		return fmt.Errorf("conference base url is required")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse conference base url: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("conference base url must use http or https")
	}

	if parsedURL.Host == "" {
		return fmt.Errorf("conference base url host is required")
	}

	return nil
}

func (c *Config) Validate() error {
	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("validate http config: %w", err)
	}

	if err := c.PG.Validate(); err != nil {
		return fmt.Errorf("validate postgres config: %w", err)
	}

	if err := c.Logger.Validate(); err != nil {
		return fmt.Errorf("validate logger config: %w", err)
	}

	if err := c.Auth.Validate(); err != nil {
		return fmt.Errorf("validate auth config: %w", err)
	}

	if err := c.Conference.Validate(); err != nil {
		return fmt.Errorf("validate conference config: %w", err)
	}

	return nil
}

func New() (*Config, error) {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}
