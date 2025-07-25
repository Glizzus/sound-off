package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// SoundCronStreamJob represents a job that will be sent to a worker
// for execution. It should container everything necessary
// for the worker to execute a job so that the worker
// can be as lightweight as possible.
type SoundCronStreamJob struct {
	// SoundCronID is the ID of the SoundCron that this
	// job is tied to.
	SoundCronID string

	// Name is the user-facing name of the SoundCron.
	Name string

	// GuildID is the Discord ID of the Guild that
	// the worker should execute the job on.
	GuildID string

	// RunTime is the exact point in time that the worker
	// should execute the critical section of the job.
	RunTime time.Time

	// TargetChannelID is the Discord ID of the voice
	// channel that the worker should join or interact
	// with in executing the job.
	TargetChannelID string
}

// JobSender is an interface for anything that can handle
// multiple SoundCronStreamJob instances.
// A JobSender implementation could put the jobs in a job queue,
// print the jobs, or execute the jobs themselves.
type JobSender interface {
	HandleJobs(ctx context.Context, jobs ...SoundCronStreamJob) error
}

// PrintingJobSender is a JobSender implementation that just
// logs provided SoundCronStreamJob instances.
// This is intended for development and debugging.
type PrintingJobSender struct{}

func (h *PrintingJobSender) HandleJobs(ctx context.Context, jobs ...SoundCronStreamJob) error {
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

var _ JobSender = (*PrintingJobSender)(nil)

// RedisJobSender is a JobSender implementation that sends
// SoundCronStreamJob instances to a Redis stream.
// The idea is that a subscriber to the stream will pick up these
// jobs and execute them.
type RedisJobSender struct {
	client *redis.Client
}

// NewRedisJobSender constructs a new RedisJobSender instance using the given
// Redis client instance. This will create the Redis stream for jobs
// if it doesn't exist.
// If there is a failure to create the Redis stream, this returns an error.
func NewRedisJobSender(client *redis.Client) (*RedisJobSender, error) {
	err := client.XGroupCreateMkStream(context.Background(), "soundcron_jobs", "soundcron_streaming_group", "$").Err()
	if err != nil && err != redis.Nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return nil, err
	}

	return &RedisJobSender{client: client}, nil
}

func (h *RedisJobSender) HandleJobs(ctx context.Context, jobs ...SoundCronStreamJob) error {
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

var _ JobSender = (*RedisJobSender)(nil)

// BlacklistAdder is an interface that defines behavior
// for adding SoundCron IDs to a blacklist.
type BlacklistAdder interface {
	AddToBlacklist(ctx context.Context, soundCronID string) error
}

// BlacklistChecker is an interface that defines behavior
// for checking if a SoundCron ID exists in a blacklist.
type BlacklistChecker interface {
	IsBlacklisted(ctx context.Context, soundCronID string) (bool, error)
}

// RedisBlackListHandler contains behavior related to manipulating
// blacklists for SoundCron IDs. It is backed by Redis.
type RedisBlacklistHandler struct {
	client *redis.Client
}

func NewRedisBlacklistHandler(client *redis.Client) *RedisBlacklistHandler {
	return &RedisBlacklistHandler{
		client: client,
	}
}

// SoundCronJobBlacklistKey returns the Redis key for marking a SoundCron ID as blacklisted.
func SoundCronJobBlacklistKey(soundCronID string) string {
	return fmt.Sprintf("soundcron:job:%s:blacklist", soundCronID)
}

func (h *RedisBlacklistHandler) AddToBlacklist(ctx context.Context, soundCronID string) error {
	key := SoundCronJobBlacklistKey(soundCronID)
	ttl := 24 * time.Hour
	_, err := h.client.Set(ctx, key, "1", ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to add soundCronID %s to blacklist: %w", soundCronID, err)
	}
	return nil
}

func (h *RedisBlacklistHandler) IsBlacklisted(ctx context.Context, soundCronID string) (bool, error) {
	key := SoundCronJobBlacklistKey(soundCronID)
	val, err := h.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		return false, fmt.Errorf("failed to check blacklist for soundCronID %s: %w", soundCronID, err)
	}
	return val == "1", nil
}

var _ BlacklistAdder = (*RedisBlacklistHandler)(nil)
var _ BlacklistChecker = (*RedisBlacklistHandler)(nil)

type MemoryBlacklistAdder struct {
	blacklist map[string]struct{}
}

func NewMemoryBlacklistAdder() *MemoryBlacklistAdder {
	return &MemoryBlacklistAdder{
		blacklist: make(map[string]struct{}),
	}
}

func (m *MemoryBlacklistAdder) AddToBlacklist(ctx context.Context, soundCronID string) error {
	m.blacklist[soundCronID] = struct{}{}
	return nil
}

func (m *MemoryBlacklistAdder) IsBlacklisted(ctx context.Context, soundCronID string) (bool, error) {
	_, exists := m.blacklist[soundCronID]
	return exists, nil
}

type JobReceiver interface {
	ReceiveJobs(ctx context.Context) ([]SoundCronStreamJob, error)
}

type RedisJobReceiver struct {
	client   *redis.Client
	consumer string
}

func NewRedisJobReceiver(client *redis.Client, consumer string) *RedisJobReceiver {
	return &RedisJobReceiver{client: client, consumer: consumer}
}

func (r *RedisJobReceiver) ReceiveJobs(ctx context.Context) ([]SoundCronStreamJob, error) {
	const (
		streamName = "soundcron_jobs"
		groupName  = "soundcron_streaming_group"
	)

	var (
		jobs []SoundCronStreamJob
		errs []error
	)

	streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    groupName,
		Consumer: r.consumer,
		Streams:  []string{streamName, ">"},
		Block:    0,
		Count:    100,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to read from Redis stream: %w", err)
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			job, err := parseStreamJob(msg)
			if err != nil {
				errs = append(errs, fmt.Errorf("message %s: %w", msg.ID, err))
				continue
			}

			jobs = append(jobs, job)

			if _, err := r.client.XAck(ctx, streamName, groupName, msg.ID).Result(); err != nil {
				errs = append(errs, fmt.Errorf("failed to acknowledge message %s: %w", msg.ID, err))
			}
		}
	}

	return jobs, errors.Join(errs...)
}

func parseStreamJob(msg redis.XMessage) (SoundCronStreamJob, error) {
	getString := func(key string) (string, error) {
		v, ok := msg.Values[key]
		if !ok {
			return "", fmt.Errorf("missing key %q", key)
		}
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("key %q is not a string", key)
		}
		return s, nil
	}

	rawRunAt, err := getString("runAt")
	if err != nil {
		return SoundCronStreamJob{}, err
	}
	runAt, err := time.Parse(time.RFC3339, rawRunAt)
	if err != nil {
		return SoundCronStreamJob{}, fmt.Errorf("invalid runAt time: %w", err)
	}

	jobName, err := getString("jobName")
	if err != nil {
		return SoundCronStreamJob{}, err
	}
	soundCronID, err := getString("soundCronID")
	if err != nil {
		return SoundCronStreamJob{}, err
	}
	guildID, err := getString("guildID")
	if err != nil {
		return SoundCronStreamJob{}, err
	}
	targetChannelID, err := getString("targetChannelID")
	if err != nil {
		return SoundCronStreamJob{}, err
	}

	return SoundCronStreamJob{
		Name:            jobName,
		SoundCronID:     soundCronID,
		GuildID:         guildID,
		RunTime:         runAt,
		TargetChannelID: targetChannelID,
	}, nil
}

var _ JobReceiver = (*RedisJobReceiver)(nil)
