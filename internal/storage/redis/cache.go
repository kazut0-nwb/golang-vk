package redisstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{
		client: client,
	}
}

// Get получает значение из Redis.
// found=false означает, что ключ отсутствует.
func (c *Cache) Get(
	ctx context.Context,
	key string,
) (value string, found bool, err error) {
	result, err := c.client.Get(
		ctx,
		searchCachePrefix+key,
	).Result()

	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}

	if err != nil {
		return "", false, fmt.Errorf(
			"ошибка чтения поискового кэша: %w",
			err,
		)
	}

	return result, true, nil
}

// Set сохраняет успешный поисковый ответ с заданным TTL.
func (c *Cache) Set(
	ctx context.Context,
	key string,
	value string,
	ttl time.Duration,
) error {
	if ttl <= 0 {
		return errors.New("TTL кэша должен быть положительным")
	}

	if err := c.client.Set(
		ctx,
		searchCachePrefix+key,
		value,
		ttl,
	).Err(); err != nil {
		return fmt.Errorf(
			"ошибка записи поискового кэша: %w",
			err,
		)
	}

	return nil
}
