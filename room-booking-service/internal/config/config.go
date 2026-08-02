package config

import (
	"fmt"
	"net"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	PG struct {
		Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
		Port     string `env:"POSTGRES_PORT" envDefault:"5432"`
		DB       string `env:"POSTGRES_DB" envDefault:"meeting_rooms"`
		User     string `env:"POSTGRES_USER" envDefault:"meeting_rooms_user"`
		Password string `env:"POSTGRES_PASSWORD" envDefault:"12345"`
	}
}

func (c *Config) ConstructPostgresURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		c.PG.User,
		c.PG.Password,
		net.JoinHostPort(c.PG.Host, c.PG.Port),
		c.PG.DB,
	)
}

func New() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)
	return &cfg, err
}
