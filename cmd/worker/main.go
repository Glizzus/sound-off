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

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateway"
	"github.com/glizzus/sound-off/internal/config"
	"github.com/glizzus/sound-off/internal/opus"
	"github.com/glizzus/sound-off/internal/schedule"
	"github.com/glizzus/sound-off/internal/voice"
	disgoVoice "github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/godave/golibdave"
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

	client, err := disgo.New(
		discordConfig.Token,
		bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentGuildVoiceStates)),	
		bot.WithVoiceManagerConfigOpts(
			disgoVoice.WithDaveSessionCreateFunc(golibdave.NewSession),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create discord session: %w", err)
	}

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

				endpoint := "http://" + minioEndpoint + "/soundoff/sound-off/opus/" + job.SoundCronID
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
						"failed to preload opus file",
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
						"failed to preload opus file",
						slog.String("soundCronID", job.SoundCronID),
					)
					return
				}
				defer respBody.Close()
				reader := opus.NewFrameReader(respBody)
				err := voice.WithVoiceChannel(ctx, client.VoiceManager, job.GuildID, job.TargetChannelID, func(vc disgoVoice.Conn) error {
					return opus.StreamToVoice(reader, vc)
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
