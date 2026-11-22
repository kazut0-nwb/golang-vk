package redisstore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

func NewClient(
	ctx context.Context,
	addr string,
) (*redis.Client, error) {
	addr = strings.TrimSpace(addr)

	if addr == "" {
		return nil, errors.New("REDIS_ADDR не задан")
	}

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()

		return nil, fmt.Errorf(
			"не удалось подключиться к Redis: %w",
			err,
		)
	}

	return client, nil
}
