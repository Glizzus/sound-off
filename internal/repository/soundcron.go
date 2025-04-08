package repository

import (
	"context"
	"fmt"

	"github.com/glizzus/sound-off/internal/schedule"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SoundCron struct {
	ID       string
	Name     string
	GuildID  string
	Cron     string
	FileSize int64
}

type SoundCronLister interface {
	List(ctx context.Context, guildID string) ([]SoundCron, error)
}

type SoundCronPersister interface {
	Save(ctx context.Context, soundCron SoundCron) error
}

type SoundCronRepository interface {
	SoundCronPersister
	SoundCronLister
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

func (r *PostgresSoundCronRepository) Save(ctx context.Context, soundCron SoundCron) error {
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

	next5Times, err := schedule.NextRunTimes(soundCron.Cron, 5)
	if err != nil {
		return fmt.Errorf("failed to get next run times: %w", err)
	}

	const soundCronJobsQuery = `
	INSERT INTO soundcron_job (soundcron_id, run_time)
	SELECT $1, unnest($2::timestamp[])
	ON CONFLICT (soundcron_id, run_time) DO NOTHING
	`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			fmt.Printf("failed to rollback transaction: %v\n", err)
		}
	}()

	_, err = tx.Exec(ctx, soundCronQuery, SoundCronToRowParams(soundCron)...)
	if err != nil {
		return fmt.Errorf("failed to execute sound cron query: %w", err)
	}

	_, err = tx.Exec(ctx, soundCronJobsQuery, soundCron.ID, next5Times)
	if err != nil {
		return fmt.Errorf("failed to execute sound cron jobs query: %w", err)
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

var _ SoundCronRepository = (*PostgresSoundCronRepository)(nil)
