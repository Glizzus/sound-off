package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/glizzus/sound-off/internal/config"
	"github.com/glizzus/sound-off/internal/dca"
	"github.com/glizzus/sound-off/internal/handler"
	"github.com/glizzus/sound-off/internal/schedule"
	"github.com/glizzus/sound-off/internal/voice"
	"github.com/glizzus/sound-off/internal/worker"
	"github.com/redis/go-redis/v9"
)

func getLogAttrs(job worker.SoundCronStreamJob) []any {
	return []any{
		"soundCronID", job.SoundCronID,
		"jobName", job.Name,
		"guildID", job.GuildID,
		"runAt", job.RunTime.Format("2006-01-02 15:04:05"),
		"targetChannelID", job.TargetChannelID,
	}
}

var dryRun = flag.Bool("dry-run", false, "Do not use Discord, just print job info to terminal")

func runWorkerForever() error {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	if err := config.LoadEnv(); err != nil {
		if os.IsNotExist(err) {
			slog.Warn("No .env file found, continuing without it")
		} else {
			return fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	redisConfig, err := config.NewRedisConfigFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load redis config: %w", err)
	}

	discordConfig, err := config.NewDiscordConfigFromEnv()
	if err != nil {
		return fmt.Errorf("failed to load discord config: %w", err)
	}

	minioEndpoint := os.Getenv("MINIO_ENDPOINT")
	if minioEndpoint == "" {
		return fmt.Errorf("MINIO_ENDPOINT is not set")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Addr,
		Password: redisConfig.Password,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	consumer, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	session, err := handler.NewSession(discordConfig.Token, handler.Handlers{
		Ready: handler.ReadyLog,
	})
	if err != nil {
		return fmt.Errorf("failed to create discord session: %w", err)
	}
	if err := session.Open(); err != nil {
		return fmt.Errorf("failed to open discord session: %w", err)
	}
	defer func() {
		if err := session.Close(); err != nil {
			slog.Error("failed to close discord session", "error", err)
		}
	}()

	blacklistChecker := worker.NewRedisBlacklistHandler(rdb)
	jobReceiver := worker.NewRedisJobReceiver(rdb, consumer)

	for {
		jobs, err := jobReceiver.ReceiveJobs(context.Background())
		if err != nil {
			return fmt.Errorf("failed to receive jobs: %w", err)
		}

		for _, job := range jobs {
			respReady := make(chan io.ReadCloser, 1)
			preloadTime := job.RunTime.Add(-time.Second * 5)
			ctx := context.Background()

			schedule.RunAt(ctx, preloadTime, func(ctx context.Context) {
				blacklisted, err := blacklistChecker.IsBlacklisted(context.Background(), job.SoundCronID)
				if err != nil {
					slog.Error(
						"failed to check blacklist",
						slog.String("soundCronID", job.SoundCronID),
						slog.Any("error", err),
					)
					respReady <- nil
					return
				}
				if blacklisted {
					slog.Info(
						"skipping blacklisted job",
						slog.String("soundCronID", job.SoundCronID),
					)
					respReady <- nil
					return
				}

				endpoint := "http://" + minioEndpoint + "/soundoff/sound-off/dca/" + job.SoundCronID
				if *dryRun {
					slog.Info(
						"Dry run mode: job would be preloaded",
						"endpoint", endpoint,
					)
					return
				}
				req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					slog.Error(
						"failed to preload DCA file",
						slog.String("soundCronID", job.SoundCronID),
						slog.Any("error", err),
					)
					respReady <- nil
				} else {
					respReady <- resp.Body
				}
			})

			schedule.RunAt(ctx, job.RunTime, func(ctx context.Context) {
				if *dryRun {
					slog.Info(
						"Dry run mode: job would be executed",
						getLogAttrs(job)...,
					)
					return
				}
				respBody := <-respReady
				if respBody == nil {
					slog.Error(
						"failed to preload DCA file",
						slog.String("soundCronID", job.SoundCronID),
					)
					return
				}
				defer respBody.Close()
				decoder := dca.NewDecoder(respBody)
				err := voice.WithVoiceChannel(session, job.GuildID, job.TargetChannelID, func(_ *discordgo.Session, vc *discordgo.VoiceConnection) error {
					done := make(chan error, 1)
					stream := dca.NewStream(decoder, vc, done)
					err := <-done
					if err != nil {
						if err != io.EOF {
							return fmt.Errorf("failed to stream audio: %w", err)
						}
					}
					_, err = stream.Finished()
					if err != nil {
						return fmt.Errorf("failed to finish streaming: %w", err)
					}
					return nil
				})
				if err != nil {
					attrs := append(getLogAttrs(job), slog.Any("error", err))
					slog.Error(
						"failed to execute scheduled job",
						attrs...,
					)
				}
			})
		}
	}
}

func main() {
	if err := runWorkerForever(); err != nil {
		slog.Error("Worker encountered an error", slog.Any("error", err))
		os.Exit(1)
	}
}
