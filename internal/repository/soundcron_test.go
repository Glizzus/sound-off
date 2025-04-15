package repository_test

import (
	"testing"
	"time"

	"github.com/glizzus/sound-off/internal/datalayer"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestRepositorySave(t *testing.T) {
	ctx := t.Context()
	postgresContainer, err := postgres.Run(
		ctx,
		"postgres",
		postgres.WithDatabase("soundoff"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	defer func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate postgres container: %v", err)
		}
	}()

	connStr, err := postgresContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr) // Use connStr instead of calling ConnectionString() again
	if err != nil {
		t.Fatalf("failed to create postgres pool: %v", err)
	}

	defer pool.Close()

	if err := datalayer.MigratePostgres(pool); err != nil {
		t.Fatalf("failed to migrate postgres: %v", err)
	}

	repo := repository.NewPostgresSoundCronRepository(pool)

	id := "e281f5c0-c05f-423d-9add-c0ffee084f27"
	if err := repo.Save(ctx, repository.SoundCron{
		ID:      id,
		Name:    "Test SoundCron",
		GuildID: "1234567890",
		Cron:    "* * * * *",
	}); err != nil {
		t.Fatalf("failed to save SoundCron: %v", err)
	}

	t.Run("The SoundCron should be saved as a row in the database", func(t *testing.T) {
		rows, err := pool.Query(ctx, "SELECT id, soundcron_name, guild_id, cron FROM soundcron WHERE id = $1", id)
		if err != nil {
			t.Fatalf("failed to query SoundCron: %v", err)
		}
		defer rows.Close()

		if !rows.Next() {
			t.Fatalf("no rows returned")
		}

		var soundCron repository.SoundCron
		if err := rows.Scan(&soundCron.ID, &soundCron.Name, &soundCron.GuildID, &soundCron.Cron); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}

		if soundCron.ID != id || soundCron.Name != "Test SoundCron" || soundCron.GuildID != "1234567890" || soundCron.Cron != "* * * * *" {
			t.Errorf("SoundCron does not match expected values: %+v", soundCron)
		}
	})

	t.Run("The SoundCron should have upcoming jobs", func(t *testing.T) {
		rows, err := pool.Query(ctx, "SELECT id, soundcron_id, run_time FROM soundcron_job WHERE soundcron_id = $1", id)
		if err != nil {
			t.Fatalf("failed to query SoundCron jobs: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var job repository.SoundCronJobRow
			if err := rows.Scan(&job.ID, &job.SoundCronID, &job.RunTime); err != nil {
				t.Fatalf("failed to scan row: %v", err)
			}

			t.Run("SoundCron job should have the ID of the original SoundCron", func(t *testing.T) {
				if job.SoundCronID != id {
					t.Errorf("SoundCron job ID does not match original SoundCron ID: %s != %s", job.SoundCronID, id)
				}
			})

			t.Run("SoundCron job should be within [-1, 6] minutes", func(t *testing.T) {
				if job.RunTime.Before(time.Now().Add(-time.Minute)) || job.RunTime.After(time.Now().Add(6*time.Minute)) {
					t.Errorf("SoundCron job run time is out of range: %v", job.RunTime)
				}
			})
		}
	})
}
