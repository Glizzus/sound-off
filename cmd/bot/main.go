package main

import (
	"context"
	"flag"
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
	"github.com/glizzus/sound-off/internal/worker"
	"github.com/redis/go-redis/v9"
)

var dryRun = flag.Bool("dry-run", false, "Do not send jobs to Redis, just print job info to terminal")

const guildID = "517907971481534467"

func runBotForever() error {
	flag.Parse()
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

	repository := repository.NewPostgresSoundCronRepository(pool)

	minioStorage, err := datalayer.NewMinioStorageFromEnv()
	if err != nil {
		return fmt.Errorf("failed to create minio storage: %w", err)
	}

	if err := minioStorage.EnsureBucket(context.Background()); err != nil {
		return fmt.Errorf("failed to ensure minio bucket: %w", err)
	}

	var blacklistAdder worker.BlacklistAdder
	var jobHandler worker.JobSender
	if *dryRun {
		jobHandler = &worker.PrintingJobSender{}
		blacklistAdder = worker.NewMemoryBlacklistAdder()
	} else {
		redisConfig, err := config.NewRedisConfigFromEnv()
		if err != nil {
			return fmt.Errorf("failed to load Redis config: %w", err)
		}
		redisClient := redis.NewClient(&redis.Options{
			Addr: 	  redisConfig.Addr,
			Password: redisConfig.Password,
		})

		jobHandler, err = worker.NewRedisJobSender(redisClient)
		if err != nil {
			return fmt.Errorf("failed to create Redis job handler: %w", err)
		}
		blacklistAdder = worker.NewRedisBlacklistHandler(redisClient)
	}

	interactionHandler := handler.NewDiscordInteractionHandler(repository, minioStorage, blacklistAdder)

	discordConfig, err := config.NewDiscordConfigFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	session, err := handler.NewSession(discordConfig.Token, handler.Handlers{
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

				var streamJobs []worker.SoundCronStreamJob
				for _, job := range upcoming {
					channels, err := session.GuildChannels(job.GuildID)
					if err != nil {
						slog.Error("failed to get guild channels", "guildID", job.GuildID, "error", err)
						continue
					}
					maxAttendedChannel := voice.MaxAttendedChannel(channels)
					if maxAttendedChannel == nil {
						slog.Debug("no attended channels found for guild", "guildID", job.GuildID)
						continue
					}
					streamJobs = append(streamJobs, worker.SoundCronStreamJob{
						SoundCronID:     job.SoundCronID,
						Name:            job.Name,
						GuildID:         job.GuildID,
						RunTime:         job.RunTime,
						TargetChannelID: maxAttendedChannel.ID,
					})
				}

				go jobHandler.HandleJobs(context.Background(), streamJobs...)
				// Look into batching here (or a more sophisticated solution)
				for _, job := range upcoming {
					repository.Refresh(context.Background(), job.SoundCronID)
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
