package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/glizzus/sound-off/internal/config"
	"github.com/glizzus/sound-off/internal/datalayer"
	"github.com/glizzus/sound-off/internal/handler"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/glizzus/sound-off/internal/voice"
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

	repository := repository.NewPostgresSoundCronRepository(pool)

	minioStorage, err := datalayer.NewMinioStorageFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create minio storage: %w", err)
	}

	if err := minioStorage.EnsureBucket(context.Background()); err != nil {
		return fmt.Errorf("failed to ensure minio bucket: %w", err)
	}

	interactionHandler := handler.MakeInteractionCreateHandler(repository, minioStorage)

	session, err := handler.NewSession(config.Token, handler.Handlers{
		Ready:             handler.ReadyLog,
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

	ticker := time.NewTicker(27 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				upcoming, err := repository.Pull(context.Background(), time.Now().Add(time.Minute))
				if err != nil {
					slog.Error("failed to pull soundcrons", "error", err)
					continue
				}
				for _, sc := range upcoming {
					channels, err := session.GuildChannels(guildID)
					if err != nil {
						slog.Error("failed to get guild channels", "error", err)
						continue
					}

					guild := voice.MaxAttendedChannel(channels)
					if guild == nil {
						slog.Warn("no guild found")
						continue
					}

					log.Printf("imagine we are playing this soundcron %s in channel %s", sc.Name, guild.ID)

				}
			case <-time.After(5 * time.Minute):
			}
		}
	}()

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
