package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/glizzus/sound-off/internal/config"
	"github.com/glizzus/sound-off/internal/datalayer"
	"github.com/glizzus/sound-off/internal/handler"
)

func runBotForever() error {
	if err := config.LoadEnv(); err != nil {
		if os.IsNotExist(err) {
			slog.Warn("No .env file found, continuing without it")
		} else {
			return fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	pool, err := datalayer.NewPostgresPoolFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err := datalayer.MigratePostgres(pool); err != nil {
		return fmt.Errorf("failed to migrate postgres: %w", err)
	}

	config, err := config.NewDiscordConfigFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	interactionHandler := handler.MakeInteractionCreateHandler()

	session, err := handler.NewSession(config.Token, handler.Handlers{
		Ready:         handler.ReadyLog,
		InteractionCreate: interactionHandler,
	})
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	if err := session.Open(); err != nil {
		return fmt.Errorf("failed to open session: %w", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			slog.Warn("failed to close session", "error", err)
		}
	}()

	// TODO: Commit to global commands
	const guildID = "517907971481534467"
	if err := handler.EstablishCommands(session, guildID); err != nil {
		return fmt.Errorf("failed to establish commands: %w", err)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop
	return nil
}

func main() {
	if err := runBotForever(); err != nil {
		log.Fatalf("failed to run bot: %v", err)
	}
}
