package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/glizzus/sound-off/internal/config"
	"github.com/glizzus/sound-off/internal/datalayer"
	"github.com/glizzus/sound-off/internal/handler"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/glizzus/sound-off/internal/schedule"
	"github.com/glizzus/sound-off/internal/voice"
	"github.com/jogramming/dca"
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

					channel := voice.MaxAttendedChannel(channels)
					if channel == nil {
						slog.Warn("no channel found")
						continue
					}

					job := schedule.ScheduledJob{
						RunAt: sc.RunTime,
						Execute: func() {
							err = voice.WithVoiceChannel(session, channel.GuildID, channel.ID, func(s *discordgo.Session, vc *discordgo.VoiceConnection) error {
								// TODO: Dynamicize the endpoint
								url := "http://localhost:9000/" + sc.SoundCronID
								audioSession, err := voice.StreamDCAOnTheFly(context.Background(), url)
								if err != nil {
									return fmt.Errorf("unable to stream dca on the fly: %w", err)
								}

								done := make(chan error, 1)
								_ = dca.NewStream(audioSession, vc, done)
								err = <-done
								if err != nil && err != io.EOF {
									return fmt.Errorf("error occurred while playing sound: %w", err)
								}

								return nil
							})
							if err != nil {
								slog.Error("failed to play sound", "error", err)
							}
						},
					}
					job.Schedule()
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
