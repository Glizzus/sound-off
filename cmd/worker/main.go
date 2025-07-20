package main

import (
	"context"
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

func runWorkerForever() error {
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

	rdb := redis.NewClient(&redis.Options{
		Addr: redisConfig.Addr,
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

	dcaStreamer := voice.NewFFmpegDCAStreamer(&voice.HTTPURLReader{Client: http.DefaultClient})

	for {
		entries, err := rdb.XReadGroup(context.Background(), &redis.XReadGroupArgs{
			Group:    "soundcron_streaming_group",
			Consumer: consumer,
			Streams:  []string{"soundcron_jobs", ">"},
			Block:    0, // Block forever until a new job arrives
		}).Result()
		if err != nil {
			slog.Error("failed to read from redis stream", slog.Any("error", err))
			time.Sleep(2 * time.Second) // Prevent tight error loop
			continue
		}

		for _, stream := range entries {
			for _, msg := range stream.Messages {
				rawRunAt, ok := msg.Values["runAt"].(string)
				if !ok {
					slog.Error("runAt field is missing or not a string", slog.String("messageID", msg.ID))
					continue
				}

				runAt, err := time.Parse(time.RFC3339, rawRunAt)
				if err != nil {
					slog.Error("failed to parse runAt time", slog.String("runAt", rawRunAt), slog.String("messageID", msg.ID), slog.Any("error", err))
					continue
				}

				job := worker.SoundCronStreamJob{
					Name:            msg.Values["jobName"].(string),
					SoundCronID:     msg.Values["soundCronID"].(string),
					GuildID:         msg.Values["guildID"].(string),
					RunTime:         runAt,
					TargetChannelID: msg.Values["targetChannelID"].(string),
				}

				fmt.Printf("%+v\n", job)

				scheduledJob := schedule.ScheduledJob{
					RunAt: job.RunTime,
					Execute: func() {
						err := voice.WithVoiceChannel(session, job.GuildID, job.TargetChannelID, func(_ *discordgo.Session, vc *discordgo.VoiceConnection) error {
							slog.Info("joining voice channel", slog.String("guildID", job.GuildID), slog.String("channelID", job.TargetChannelID))
							endpoint := "http://minio:9000/soundoff/" + job.SoundCronID
							ctx := context.Background()
							audioSession, err := dcaStreamer.StreamDCAOnTheFly(ctx, endpoint)
							if err != nil {
								return fmt.Errorf("failed to stream DCA: %w", err)
							}
							defer audioSession.Cleanup()

							done := make(chan error, 1)
							stream := dca.NewStream(audioSession, vc, done)
							err = <-done
							if err != nil {
								if err != io.EOF {
									return fmt.Errorf("failed to stream audio: %w", err)
								}
							}

							_, err = stream.Finished()
							if err != nil {
								return fmt.Errorf("failed to finish streaming: %w", err)
							}

							err = audioSession.Error()
							if err != nil {
								return fmt.Errorf("audio session error: %w", err)
							}

							return nil
						})
						if err != nil {
							slog.Error(
								"failed to execute scheduled job",
								slog.String("jobName", job.Name),
								slog.String("soundCronID", job.SoundCronID),
								slog.String("guildID", job.GuildID),
								slog.String("runAt", job.RunTime.Format(time.RFC3339)),
								slog.String("targetChannelID", job.TargetChannelID),
								slog.Any("error", err),
							)
						}
					},
				}
				scheduledJob.Schedule()
			}
		}
	}
	// unreachable
}

func main() {
	if err := runWorkerForever(); err != nil {
		slog.Error("Worker encountered an error", slog.Any("error", err))
		os.Exit(1)
	}
}
