package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	const (
		pingTimeout   = 2 * time.Second
		retryInterval = 250 * time.Millisecond
	)

	ticker := time.NewTicker(retryInterval)
	defer ticker.Stop()

	var lastErr error

	for {
		// Каждая попытка имеет собственный таймаут.
		pingCtx, pingCancel := context.WithTimeout(ctx, pingTimeout)
		pingErr := pool.Ping(pingCtx)
		pingCancel()

		if pingErr == nil {
			return pool, nil
		}

		lastErr = pingErr

		if ctx.Err() != nil {
			pool.Close()

			return nil, fmt.Errorf(
				"не удалось подключиться к PostgreSQL: %w; последняя ошибка: %v",
				ctx.Err(),
				lastErr,
			)
		}

		log.Printf(
			"PostgreSQL пока недоступен, повторная попытка: %v",
			lastErr,
		)

		select {
		case <-ctx.Done():
			pool.Close()

			return nil, fmt.Errorf(
				"не удалось подключиться к PostgreSQL: %w; последняя ошибка: %v",
				ctx.Err(),
				lastErr,
			)

		case <-ticker.C:
		}
	}
}
