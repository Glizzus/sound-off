package e2e

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/glizzus/sound-off/internal/datalayer"
	"github.com/glizzus/sound-off/internal/generator"
	"github.com/glizzus/sound-off/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
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
	postgresOnce      sync.Once
	postgresContainer *postgres.PostgresContainer
	postgresConnStr   string
	postgresStartErr  error
	pool              *pgxpool.Pool
	postgresWG        sync.WaitGroup
)

// UsePostgres signals that the test is using Postgres as its database.
// This will either provision or reuse a Postgres container for the test.
// Do not expect a clean state in the database; it is shared across tests
// to simulate real-world usage.
func UsePostgres(t *testing.T) string {
	t.Helper()

	postgresOnce.Do(func() {
		ctx := context.Background()
		postgresContainer, postgresStartErr = postgres.Run(
			ctx,
			"postgres@sha256:3962158596daaef3682838cc8eb0e719ad1ce520f88e34596ce8d5de1b6330a1",
			postgres.WithDatabase("soundoff"),
			postgres.WithUsername("user"),
			postgres.WithPassword("password"),
			postgres.BasicWaitStrategies(),
		)
		if postgresStartErr != nil {
			return
		}
		postgresConnStr, postgresStartErr = postgresContainer.ConnectionString(ctx)
		if postgresStartErr != nil {
			return
		}

		pool, postgresStartErr = pgxpool.New(ctx, postgresConnStr)
		if postgresStartErr != nil {
			return
		}
		defer pool.Close()

		postgresStartErr = datalayer.MigratePostgres(pool)
	})

	if postgresStartErr != nil {
		t.Fatalf("failed to start postgres container: %v", postgresStartErr)
	}
	t.Logf("Postgres container being used by test %s", t.Name())
	postgresWG.Add(1)
	t.Cleanup(postgresWG.Done)

	return postgresConnStr
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
	postgresWG.Wait()
	log.Printf("Terminating Postgres container")
	if postgresContainer != nil {
		err := postgresContainer.Terminate(context.Background())
		if err != nil {
			fmt.Printf("failed to terminate postgres container: %v", err)
		}
	}
}

var (
	redisOnce      sync.Once
	redisContainer *redis.RedisContainer
	redisConnStr   string
	redisStartErr  error

	redisWG sync.WaitGroup
)

// UseRedis signals that the test will use Redis.
// It starts or reuses a Redis container for the test.
// This function reuses the Redis container across tests to simulate real-world usage.
// It returns the Redis connection string for the test to use.
func UseRedis(t *testing.T) string {
	t.Helper()
	redisOnce.Do(func() {
		ctx := context.Background()
		redisContainer, redisStartErr = redis.Run(
			ctx,
			"redis",
		)
		if redisStartErr != nil {
			return
		}
		redisConnStr, redisStartErr = redisContainer.ConnectionString(ctx)
		if redisStartErr != nil {
			return
		}
	})
	if redisStartErr != nil {
		t.Fatalf("failed to start redis container: %v", redisStartErr)
	}
	t.Logf("Redis container being used by test %s", t.Name())
	redisWG.Add(1)
	t.Cleanup(redisWG.Done)
	return redisConnStr
}

func TerminateRedisForE2E() {
	redisWG.Wait()
	log.Printf("Terminating Redis container")
	if redisContainer != nil {
		err := redisContainer.Terminate(context.Background())
		if err != nil {
			fmt.Printf("failed to terminate redis container: %v", err)
		}
	}
}
