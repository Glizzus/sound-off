package datalayer

import (
	"context"
	"embed"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	pgxMigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// NewPostgresPoolFromEnv constructs a new ins
func NewPostgresPoolFromEnv() (*pgxpool.Pool, error) {
	ctx := context.Background()
	return pgxpool.New(ctx, "")
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func MigratePostgres(pool *pgxpool.Pool) (err error) {
	db := stdlib.OpenDBFromPool(pool)
	defer func() {
		if cerr := db.Close(); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()

	driver, derr := pgxMigrate.WithInstance(db, &pgxMigrate.Config{})
	if derr != nil {
		return derr
	}

	src, serr := iofs.New(migrationsFS, "migrations")
	if serr != nil {
		return serr
	}

	m, merr := migrate.NewWithInstance(
		"iofs",
		src,
		"pgx5",
		driver,
	)
	if merr != nil {
		return merr
	}

	defer func() {
		srcErr, dbErr := m.Close()
		err = errors.Join(err, srcErr, dbErr)
	}()

	if upErr := m.Up(); upErr != nil && !errors.Is(upErr, migrate.ErrNoChange) {
		return upErr
	}
	return nil
}
