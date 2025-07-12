package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
)

type DiscordConfig struct {
	Token          string `env:"DISCORD_TOKEN, required"`
	guildID        string `env:"DISCORD_GUILD_ID"`
	runBotGlobally bool   `env:"DISCORD_RUN_BOT_GLOBALLY"`
	ClientID       string `env:"DISCORD_CLIENT_ID, required"`
}

func (c *DiscordConfig) GuildID() string {
	if c.runBotGlobally {
		return ""
	}
	return c.guildID
}

func NewDiscordConfigFromEnv() (*DiscordConfig, error) {
	var cfg DiscordConfig
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, err
	}
	if cfg.guildID == "" && !cfg.runBotGlobally {
		return nil, fmt.Errorf("DISCORD_GUILD_ID must be set if DISCORD_RUN_BOT_GLOBALLY is false")
	}

	return &cfg, nil
}
