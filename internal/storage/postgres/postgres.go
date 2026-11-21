package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, errors.New("DATABASE_URL не задан")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка конфигурации PostgreSQL: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 1

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать пул PostgreSQL: %w", err)
	}

	// Повторяем подключение, если PostgreSQL временно недоступен.
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error

	for {
		lastErr = pool.Ping(ctx)

		if lastErr == nil {
			return pool, nil
		}

		select {
		case <-ctx.Done():
			pool.Close()

			return nil, fmt.Errorf(
				"не удалось подключиться к PostgreSQL: %w; последняя ошибка: %v",
				ctx.Err(),
				lastErr,
			)

		case <-ticker.C:
			// Повторяем попытку подключения.
		}
	}
}
