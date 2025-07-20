package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type SoundCronStreamJob struct {
	SoundCronID string
	Name        string
	GuildID     string
	RunTime     time.Time

	TargetChannelID string
}

type JobHandler interface {
	HandleJobs(ctx context.Context, jobs ...SoundCronStreamJob) error
}

type PrintingJobHandler struct{}

func (h *PrintingJobHandler) HandleJobs(ctx context.Context, jobs ...SoundCronStreamJob) error {
	for _, job := range jobs {
		slog.InfoContext(
			ctx,
			"Handling SoundCron Job",
			slog.String("soundCronID", job.SoundCronID),
			slog.String("jobName", job.Name),
			slog.String("guildID", job.GuildID),
			slog.String("runAt", job.RunTime.Format("2006-01-02 15:04:05")),
			slog.String("targetChannelID", job.TargetChannelID),
		)
	}
	return nil
}

type RedisJobHandler struct {
	client *redis.Client
}

func NewRedisJobHandler(client *redis.Client) (*RedisJobHandler, error) {
	err := client.XGroupCreateMkStream(context.Background(), "soundcron_jobs", "soundcron_streaming_group", "$").Err()
	if err != nil && err != redis.Nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, err
	}

	return &RedisJobHandler{client: client}, nil
}

func (h *RedisJobHandler) HandleJobs(ctx context.Context, jobs ...SoundCronStreamJob) error {
	_, err := h.client.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for _, job := range jobs {
			pipe.XAdd(ctx, &redis.XAddArgs{
				Stream: "soundcron_jobs",
				Values: map[string]any{
					"jobName":         job.Name,
					"soundCronID":     job.SoundCronID,
					"guildID":         job.GuildID,
					"runAt":           job.RunTime.Format(time.RFC3339),
					"targetChannelID": job.TargetChannelID,
				},
			})
		}
		return nil
	})
	return err
}

type BlacklistAdder interface {
	AddToBlacklist(ctx context.Context, soundCronID string) error
}

type RedisBlacklistAdder struct {
	client *redis.Client
}

func NewRedisBlacklistAdder(client *redis.Client) *RedisBlacklistAdder {
	return &RedisBlacklistAdder{client: client}
}

func (a *RedisBlacklistAdder) AddToBlacklist(ctx context.Context, soundCronID string) error {
	_, err := a.client.SAdd(ctx, "soundcron_blacklist", soundCronID).Result()
	if err != nil {
		return fmt.Errorf("failed to add soundCronID %s to blacklist: %w", soundCronID, err)
	}
	return nil
}

type MemoryBlacklistAdder struct {
	blacklist map[string]struct{}
}

func NewMemoryBlacklistAdder() *MemoryBlacklistAdder {
	return &MemoryBlacklistAdder{
		blacklist: make(map[string]struct{}),
	}
}

func (a *MemoryBlacklistAdder) AddToBlacklist(ctx context.Context, soundCronID string) error {
	a.blacklist[soundCronID] = struct{}{}
	return nil
}
