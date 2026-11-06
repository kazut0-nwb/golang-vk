package usecase

import (
	"context"
	"errors"
	"fmt"

	"my-go-app/internal/domain"
)

// MovieSynchronizer синхронизирует фильмы
// с внешнего стенда в локальную базу данных.
type MovieSynchronizer struct {
	provider MovieProvider
	storage  MovieStorage
}

func NewMovieSynchronizer(
	provider MovieProvider,
	storage MovieStorage,
) *MovieSynchronizer {
	return &MovieSynchronizer{
		provider: provider,
		storage:  storage,
	}
}

// Sync загружает отсутствующие фильмы в PostgreSQL.
func (s *MovieSynchronizer) Sync(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Определяем последний доступный фильм на стенде.
	lastMovie, err := s.findLastMovie(ctx)
	if err != nil {
		return fmt.Errorf("не удалось определить последний фильм: %w", err)
	}

	// Получаем фильмы, которые уже находятся в БД.
	existingMovies, err := s.storage.GetMovies(ctx)
	if err != nil {
		return fmt.Errorf("не удалось получить фильмы из БД: %w", err)
	}

	existingIDs := make(map[int]struct{}, len(existingMovies))

	for _, movie := range existingMovies {
		existingIDs[movie.ID] = struct{}{}
	}

	// Загружаем только отсутствующие фильмы.
	for id := 1; ; id++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if _, exists := existingIDs[id]; !exists {
			movie, err := s.provider.FetchMovie(ctx, id)

			if errors.Is(err, domain.ErrMovieNotFound) {
				// Такого ID на стенде нет.
			} else if err != nil {
				return fmt.Errorf(
					"не удалось загрузить фильм %d: %w",
					id,
					err,
				)
			} else {
				if movie == nil {
					return fmt.Errorf(
						"внешний клиент вернул пустой фильм %d",
						id,
					)
				}

				if movie.ID == 0 {
					movie.ID = id
				}

				if movie.ID != id {
					return fmt.Errorf(
						"ID полученного фильма %d не совпадает с запрошенным %d",
						movie.ID,
						id,
					)
				}

				if err := s.storage.SaveMovie(ctx, movie); err != nil {
					return fmt.Errorf(
						"не удалось сохранить фильм %d: %w",
						id,
						err,
					)
				}

				existingIDs[id] = struct{}{}
			}
		}

		if id == lastMovie.ID {
			break
		}
	}

	return nil
}

// findLastMovie определяет последний доступный фильм.
// Используется алгоритм экспоненциального поиска
// с последующим бинарным поиском.
func (s *MovieSynchronizer) findLastMovie(
	ctx context.Context,
) (*domain.Movie, error) {
	fetch := func(id int) (*domain.Movie, error) {
		movie, err := s.provider.FetchMovie(ctx, id)
		if err != nil {
			return nil, err
		}

		if movie == nil {
			return nil, fmt.Errorf(
				"внешний клиент вернул пустой фильм %d",
				id,
			)
		}

		if movie.ID == 0 {
			movie.ID = id
		}

		return movie, nil
	}

	last, err := fetch(1)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить первый фильм: %w", err)
	}

	left, right := 1, 2
	maxInt := int(^uint(0) >> 1)

	// Экспоненциальный поиск верхней границы.
	for {
		movie, err := fetch(right)

		if errors.Is(err, domain.ErrMovieNotFound) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf(
				"ошибка проверки фильма %d: %w",
				right,
				err,
			)
		}

		last = movie
		left = right

		if right > maxInt/2 {
			return nil, errors.New(
				"слишком большой идентификатор последнего фильма",
			)
		}

		right *= 2
	}

	// Бинарный поиск последнего существующего фильма.
	for right-left > 1 {
		mid := left + (right-left)/2

		movie, err := fetch(mid)

		if errors.Is(err, domain.ErrMovieNotFound) {
			right = mid
			continue
		}

		if err != nil {
			return nil, fmt.Errorf(
				"ошибка проверки фильма %d: %w",
				mid,
				err,
			)
		}

		last = movie
		left = mid
	}

	return last, nil
}
