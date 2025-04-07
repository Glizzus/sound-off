package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
)

type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST, required"`
	Port     string `env:"POSTGRES_PORT, default=5432"`
	Username string `env:"POSTGRES_USERNAME, required"`
	Password string `env:"POSTGRES_PASSWORD, required"`
	Database string `env:"POSTGRES_DATABASE, required"`
	SSLMode  string `env:"POSTGRES_SSLMODE, default=disable"`
}

func NewPostgresConfigFromEnv() (*PostgresConfig, error) {
	var cfg PostgresConfig
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}
