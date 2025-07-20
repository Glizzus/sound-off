package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
)

type RedisConfig struct {
	Addr     string `env:"REDIS_ADDR, required"`
	Password string `env:"REDIS_PASSWORD"`
}

func NewRedisConfigFromEnv() (*RedisConfig, error) {
	var cfg RedisConfig
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, err
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("REDIS_ADDR is required")
	}
	return &cfg, nil
}
