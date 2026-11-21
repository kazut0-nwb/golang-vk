package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"my-go-app/internal/domain"
	"my-go-app/internal/search"
)

const SearchCacheTTL = time.Hour

var ErrEmptySearchQuery = errors.New(
	"поисковый запрос не должен быть пустым",
)

// MovieSearchStorage предоставляет получение фильмов по ID.
type MovieSearchStorage interface {
	GetMoviesByIDs(
		ctx context.Context,
		ids []int,
	) ([]domain.Movie, error)
}

// MovieIndex предоставляет поиск ID в Redis.
type MovieIndex interface {
	SearchIDs(
		ctx context.Context,
		tokens []string,
	) ([]int, error)
}

// SearchResponseCache предоставляет операции с кэшем ответов.
type SearchResponseCache interface {
	Get(
		ctx context.Context,
		key string,
	) (string, bool, error)

	Set(
		ctx context.Context,
		key string,
		value string,
		ttl time.Duration,
	) error
}

type SearchService struct {
	storage MovieSearchStorage
	index   MovieIndex
	cache   SearchResponseCache
}

func NewSearchService(
	storage MovieSearchStorage,
	index MovieIndex,
	cache SearchResponseCache,
) *SearchService {
	return &SearchService{
		storage: storage,
		index:   index,
		cache:   cache,
	}
}

// Search выполняет поиск фильмов по названию и режиссёру.
func (s *SearchService) Search(
	ctx context.Context,
	query string,
) ([]domain.Movie, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		return nil, ErrEmptySearchQuery
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cacheKey := makeCacheKey(query)

	// 1. Сначала проверяем кэш.
	cached, found, err := s.cache.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	if found {
		var movies []domain.Movie

		if err := json.Unmarshal(
			[]byte(cached),
			&movies,
		); err != nil {
			return nil, fmt.Errorf(
				"не удалось прочитать поисковый кэш: %w",
				err,
			)
		}

		if movies == nil {
			movies = make([]domain.Movie, 0)
		}

		return movies, nil
	}

	// 2. Разбиваем запрос на слова.
	tokens := search.Tokenize(query)

	movies := make([]domain.Movie, 0)

	if len(tokens) > 0 {
		// 3. Ищем ID через обратный индекс Redis.
		ids, err := s.index.SearchIDs(ctx, tokens)
		if err != nil {
			return nil, fmt.Errorf(
				"ошибка поиска ID в Redis: %w",
				err,
			)
		}

		if len(ids) > 0 {
			// 4. Получаем полные записи из PostgreSQL.
			movies, err = s.storage.GetMoviesByIDs(ctx, ids)
			if err != nil {
				return nil, fmt.Errorf(
					"ошибка получения фильмов из PostgreSQL: %w",
					err,
				)
			}
		}
	}

	if movies == nil {
		movies = make([]domain.Movie, 0)
	}

	// 5. Кодируем полный успешный ответ.
	data, err := json.Marshal(movies)
	if err != nil {
		return nil, fmt.Errorf(
			"не удалось сериализовать результаты поиска: %w",
			err,
		)
	}

	// 6. Кэшируем ответ ровно на один час.
	if err := s.cache.Set(
		ctx,
		cacheKey,
		string(data),
		SearchCacheTTL,
	); err != nil {
		return nil, err
	}

	return movies, nil
}

// makeCacheKey создаёт отдельный ключ для каждого запроса.
func makeCacheKey(query string) string {
	sum := sha256.Sum256([]byte(query))
	return hex.EncodeToString(sum[:])
}

// Проверяем, что SearchService реализует поисковый интерфейс.
var _ MovieSearchUseCase = (*SearchService)(nil)
