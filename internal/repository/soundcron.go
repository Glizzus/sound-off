package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/glizzus/sound-off/internal/schedule"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SoundCron struct {
	ID       string
	Name     string
	GuildID  string
	Cron     string
	FileSize int64
}

type SoundCronJob struct {
	SoundCronID string
	Name        string
	GuildID     string
	RunTime     time.Time
}

type SoundCronJobRow struct {
	ID          string
	SoundCronID string
	RunTime     time.Time
}

type SoundCronLister interface {
	List(ctx context.Context, guildID string) ([]SoundCron, error)
}

type SoundCronPersister interface {
	Save(ctx context.Context, soundCron SoundCron) error
}

type SoundCronJobPuller interface {
	Pull(ctx context.Context, within time.Time) ([]SoundCronJob, error)
}

type SoundCronRefresher interface {
	Refresh(ctx context.Context, soundCronID string) error
}

type SoundCronRepository interface {
	SoundCronPersister
	SoundCronLister
	SoundCronJobPuller
	SoundCronRefresher
}

type PostgresSoundCronRepository struct {
	db *pgxpool.Pool
}

func NewPostgresSoundCronRepository(db *pgxpool.Pool) *PostgresSoundCronRepository {
	return &PostgresSoundCronRepository{db: db}
}

func SoundCronToRowParams(soundCron SoundCron) []any {
	return []any{
		soundCron.ID,
		soundCron.Name,
		soundCron.GuildID,
		soundCron.Cron,
		soundCron.FileSize,
	}
}

type pgxExecer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func (r *PostgresSoundCronRepository) Save(ctx context.Context, soundCron SoundCron) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			fmt.Printf("failed to rollback transaction: %v\n", err)
		}
	}()

	const soundCronQuery = `
	INSERT INTO soundcron (id, soundcron_name, guild_id, cron, file_size)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (id)
	DO UPDATE SET
		soundcron_name = EXCLUDED.soundcron_name,
		guild_id = EXCLUDED.guild_id,
		cron = EXCLUDED.cron,
		file_size = EXCLUDED.file_size;
	`

	_, err = tx.Exec(ctx, soundCronQuery, SoundCronToRowParams(soundCron)...)
	if err != nil {
		return fmt.Errorf("failed to execute sound cron query: %w", err)
	}

	err = doRefresh(ctx, tx, soundCron.ID, soundCron.Cron)
	if err != nil {
		return fmt.Errorf("failed to refresh sound cron: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresSoundCronRepository) List(ctx context.Context, guildID string) ([]SoundCron, error) {
	const query = `
	SELECT id, soundcron_name, guild_id, cron, file_size
	FROM soundcron
	WHERE guild_id = $1
	`
	rows, err := r.db.Query(ctx, query, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sound cron: %w", err)
	}
	defer rows.Close()

	var soundCrons []SoundCron
	for rows.Next() {
		var sc SoundCron
		if err := rows.Scan(&sc.ID, &sc.Name, &sc.GuildID, &sc.Cron, &sc.FileSize); err != nil {
			return nil, fmt.Errorf("failed to scan sound cron: %w", err)
		}
		soundCrons = append(soundCrons, sc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over rows: %w", err)
	}

	return soundCrons, nil
}

func (r *PostgresSoundCronRepository) Pull(ctx context.Context, within time.Time) ([]SoundCronJob, error) {
	const query = `
	UPDATE soundcron_job AS scj
	SET picked_up_at = now()
	FROM soundcron AS sc
	WHERE scj.soundcron_id = sc.id
		AND scj.run_time > now()
		AND scj.run_time <= $1
		AND scj.picked_up_at IS NULL
	RETURNING scj.soundcron_id, sc.soundcron_name, sc.guild_id, scj.run_time
	`

	rows, err := r.db.Query(ctx, query, within.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query sound cron: %w", err)
	}
	defer rows.Close()

	var soundCronJobs []SoundCronJob
	for rows.Next() {
		var scj SoundCronJob
		if err := rows.Scan(&scj.SoundCronID, &scj.Name, &scj.GuildID, &scj.RunTime); err != nil {
			return nil, fmt.Errorf("failed to scan sound cron job: %w", err)
		}
		soundCronJobs = append(soundCronJobs, scj)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate over rows: %w", err)
	}
	return soundCronJobs, nil
}

func doRefresh(ctx context.Context, execer pgxExecer, soundCronID, cron string) error {
	nextRunTimes, err := schedule.NextRunTimes(cron, 5)
	if err != nil {
		return fmt.Errorf("failed to get next run times: %w", err)
	}

	const query = `
	INSERT INTO soundcron_job (soundcron_id, run_time)
	SELECT $1, unnest($2::timestamp[])
	ON CONFLICT (soundcron_id, run_time) DO NOTHING
	`

	_, err = execer.Exec(ctx, query, soundCronID, nextRunTimes)
	if err != nil {
		return fmt.Errorf("failed to execute sound cron jobs query: %w", err)
	}
	return nil
}

func (r *PostgresSoundCronRepository) Refresh(ctx context.Context, soundCronID string) error {
	const getCronQuery = `
	SELECT cron
	FROM soundcron
	WHERE id = $1
	`
	var cron string
	err := r.db.QueryRow(ctx, getCronQuery, soundCronID).Scan(&cron)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("sound cron not found: %w", err)
		}
		return fmt.Errorf("failed to query sound cron: %w", err)
	}

	err = doRefresh(ctx, r.db, soundCronID, cron)
	if err != nil {
		return fmt.Errorf("failed to refresh sound cron: %w", err)
	}
	return nil
}

var _ SoundCronRepository = (*PostgresSoundCronRepository)(nil)
