package config

import (
	"context"
	"fmt"

	"github.com/sethvargo/go-envconfig"
)

type DiscordConfig struct {
	Token          string `env:"DISCORD_TOKEN, required"`
	GuildID        string `env:"DISCORD_GUILD_ID"`
	RunBotGlobally bool   `env:"DISCORD_RUN_BOT_GLOBALLY"`
	ClientID       string `env:"DISCORD_CLIENT_ID, required"`
}

func NewDiscordConfigFromEnv() (*DiscordConfig, error) {
	var cfg DiscordConfig
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		return nil, err
	}
	if cfg.GuildID == "" && !cfg.RunBotGlobally {
		return nil, fmt.Errorf("refusing to run the bot without a guild ID unless DISCORD_RUN_BOT_GLOBALLY is set to true")
	}

	return &cfg, nil
}
