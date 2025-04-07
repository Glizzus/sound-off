package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type DiscordConfig struct {
	Token    string `env:"DISCORD_TOKEN, required"`
	ClientID string `env:"DISCORD_CLIENT_ID, required"`
}

func NewDiscordConfigFromEnv() (*DiscordConfig, error) {
	var cfg DiscordConfig
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
