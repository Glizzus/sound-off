package e2e

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/glizzus/sound-off/internal/datalayer"
	"github.com/glizzus/sound-off/internal/generator"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var seedOnce sync.Once

type RandomSnowFlakeGenerator struct {
	counter uint64
}

func (g *RandomSnowFlakeGenerator) Next() (string, error) {
	const min = 1e17
	if g.counter < min {
		g.counter = min
	}
	id := atomic.AddUint64(&g.counter, 1)
	return fmt.Sprintf("%d", id), nil
}

var _ generator.Generator[string] = (*RandomSnowFlakeGenerator)(nil)

func SeedGlobalNoise(t *testing.T, repo *repository.PostgresSoundCronRepository) {
	t.Helper()
	seedOnce.Do(func() {
		uuidGen := generator.UUIDV4Generator{}
		guildIDGen := RandomSnowFlakeGenerator{}
		for i := range 100 {
			id, _ := uuidGen.Next()
			guildID, _ := guildIDGen.Next()

			soundCron := repository.SoundCron{
				ID:      id,
				Name:    fmt.Sprintf("noise-soundcron-%d", i),
				GuildID: guildID,
				Cron:    "*/5 * * * *",
			}

			err := repo.Save(t.Context(), soundCron)
			if err != nil {
				t.Fatalf("failed to save SoundCron: %v", err)
			}
		}
	})
}

var (
	once              sync.Once
	postgresContainer *postgres.PostgresContainer
	connStr           string
	startErr          error
	pool              *pgxpool.Pool
	wg                sync.WaitGroup
)

// UsePostgres signals that the test is using Postgres as its database.
// This will either provision or reuse a Postgres container for the test.
// Do not expect a clean state in the database; it is shared across tests
// to simulate real-world usage.
func UsePostgres(t *testing.T) string {
	t.Helper()

	once.Do(func() {
		ctx := context.Background()
		postgresContainer, startErr = postgres.Run(
			ctx,
			"postgres",
			postgres.WithDatabase("soundoff"),
			postgres.WithUsername("user"),
			postgres.WithPassword("password"),
			postgres.BasicWaitStrategies(),
		)
		if startErr != nil {
			return
		}
		connStr, startErr = postgresContainer.ConnectionString(ctx)
		if startErr != nil {
			return
		}

		pool, startErr = pgxpool.New(ctx, connStr)
		if startErr != nil {
			return
		}
		defer pool.Close()

		startErr = datalayer.MigratePostgres(pool)
	})

	if startErr != nil {
		t.Fatalf("failed to start postgres container: %v", startErr)
	}
	wg.Add(1)
	t.Cleanup(wg.Done)

	return connStr
}

// GetRepository creates a new PostgresSoundCronRepository for testing.
// It uses the provided connection string to connect to the database.
// It performs no modifications or migrations on the database schema.
func GetRepository(t *testing.T, connStr string) *repository.PostgresSoundCronRepository {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), connStr)
	if err != nil {
		t.Fatalf("failed to create postgres pool: %v", err)
	}

	t.Cleanup(pool.Close)
	return repository.NewPostgresSoundCronRepository(pool)
}

func TerminatePostgresForE2E() {
	wg.Wait()
	if postgresContainer != nil {
		err := postgresContainer.Terminate(context.Background())
		if err != nil {
			fmt.Printf("failed to terminate postgres container: %v", err)
		}
	}
}
