package redisstore

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"my-go-app/internal/domain"
	"my-go-app/internal/search"

	"github.com/redis/go-redis/v9"
)

const (
	indexPrefix       = "movies:index:word:"
	indexReadyKey     = "movies:index:ready"
	searchCachePrefix = "movies:search:cache:"
)

type Index struct {
	client *redis.Client
}

func NewIndex(client *redis.Client) *Index {
	return &Index{
		client: client,
	}
}

// Rebuild полностью перестраивает обратный индекс
// по всем фильмам, полученным из PostgreSQL.
func (i *Index) Rebuild(
	ctx context.Context,
	movies []domain.Movie,
) error {
	if i == nil || i.client == nil {
		return errors.New("Redis client не инициализирован")
	}

	// word -> set of movie IDs.
	postings := make(map[string]map[string]struct{})

	for _, movie := range movies {
		if err := ctx.Err(); err != nil {
			return err
		}

		movieID := strconv.Itoa(movie.ID)

		text := movie.Title + " " + movie.Director
		tokens := search.Tokenize(text)

		for _, token := range tokens {
			if postings[token] == nil {
				postings[token] = make(map[string]struct{})
			}

			postings[token][movieID] = struct{}{}
		}
	}

	// Удаляем старый индекс и старый кэш.
	if err := i.deleteByPattern(ctx, "movies:index:*"); err != nil {
		return fmt.Errorf("не удалось удалить старый индекс: %w", err)
	}

	if err := i.deleteByPattern(
		ctx,
		searchCachePrefix+"*",
	); err != nil {
		return fmt.Errorf("не удалось очистить кэш поиска: %w", err)
	}

	// Записываем новый индекс.
	pipe := i.client.Pipeline()

	for token, ids := range postings {
		members := make([]interface{}, 0, len(ids))

		for id := range ids {
			members = append(members, id)
		}

		key := indexPrefix + token

		pipe.SAdd(ctx, key, members...)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("не удалось записать индекс Redis: %w", err)
	}

	// Маркер готовности записывается только после построения индекса.
	// TTL = 0 означает отсутствие срока действия.
	if err := i.client.Set(
		ctx,
		indexReadyKey,
		"1",
		0,
	).Err(); err != nil {
		return fmt.Errorf("не удалось установить маркер индекса: %w", err)
	}

	return nil
}

// SearchIDs возвращает ID фильмов, содержащих все переданные слова.
// Используется логика AND через Redis SINTER.
func (i *Index) SearchIDs(
	ctx context.Context,
	tokens []string,
) ([]int, error) {
	result := make([]int, 0)

	if len(tokens) == 0 {
		return result, nil
	}

	// Проверяем, что индекс построен.
	ready, err := i.client.Exists(
		ctx,
		indexReadyKey,
	).Result()

	if err != nil {
		return nil, fmt.Errorf("не удалось проверить индекс: %w", err)
	}

	if ready == 0 {
		return nil, errors.New("обратный индекс Redis не построен")
	}

	keys := make([]string, 0, len(tokens))

	for _, token := range tokens {
		keys = append(keys, indexPrefix+token)
	}

	// Пересечение множеств.
	ids, err := i.client.SInter(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("ошибка поиска в Redis: %w", err)
	}

	for _, id := range ids {
		movieID, err := strconv.Atoi(id)
		if err != nil {
			return nil, fmt.Errorf(
				"некорректный ID фильма в индексе: %s: %w",
				id,
				err,
			)
		}

		result = append(result, movieID)
	}

	return result, nil
}

// deleteByPattern удаляет все ключи,
// соответствующие заданному шаблону.
func (i *Index) deleteByPattern(
	ctx context.Context,
	pattern string,
) error {
	var cursor uint64

	keys := make([]string, 0)

	for {
		batch, nextCursor, err := i.client.Scan(
			ctx,
			cursor,
			pattern,
			500,
		).Result()

		if err != nil {
			return err
		}

		keys = append(keys, batch...)

		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return nil
	}

	if err := i.client.Del(ctx, keys...).Err(); err != nil {
		return err
	}

	return nil
}
