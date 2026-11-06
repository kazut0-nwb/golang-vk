package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	appmigrations "my-go-app/migrations"
)

// RunMigrations применяет все неприменённые миграции PostgreSQL.
func RunMigrations(
	ctx context.Context,
	pool *pgxpool.Pool,
) error {
	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf(
			"не удалось проверить соединение для миграций: %w",
			err,
		)
	}

	goose.SetBaseFS(appmigrations.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf(
			"не удалось настроить Goose: %w",
			err,
		)
	}

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf(
			"не удалось применить миграции: %w",
			err,
		)
	}

	return nil
}
