package usecase

import (
	"context"
	"errors"
	"fmt"

	"my-go-app/internal/domain"
)

// MovieProvider описывает зависимость от источника фильмов.
type MovieProvider interface {
	FetchMovie(ctx context.Context, id int) (*domain.Movie, error)
}

// MovieStorage описывает операции с локальным хранилищем фильмов.
type MovieStorage interface {
	SaveMovie(ctx context.Context, movie *domain.Movie) error
	GetMovieByID(ctx context.Context, id int) (*domain.Movie, error)
	GetMovies(ctx context.Context) ([]domain.Movie, error)
	GetLastMovie(ctx context.Context) (*domain.Movie, error)
}

type MovieError struct {
	ID      int    `json:"id"`
	Message string `json:"message"`
	Cause   error  `json:"-" swaggerignore:"true"`
}

type RangeResult struct {
	Movies []domain.Movie `json:"movies"`
	Errors []MovieError   `json:"errors"`
}

type Service struct {
	storage MovieStorage
}

func NewService(storage MovieStorage) *Service {
	return &Service{
		storage: storage,
	}
}

func ValidateRange(from, to int) error {
	if from < 1 || to < 1 || from > to {
		return fmt.Errorf(
			"%w: from=%d, to=%d; требуется 1 <= from <= to",
			domain.ErrInvalidRange, from, to,
		)
	}

	return nil
}

// LoadMovies получает фильмы из локального хранилища
// в заданном диапазоне ID.
func (s *Service) LoadMovies(
	ctx context.Context,
	from, to int,
) (RangeResult, error) {
	if err := ValidateRange(from, to); err != nil {
		return RangeResult{}, err
	}

	result := RangeResult{
		Movies: make([]domain.Movie, 0),
		Errors: make([]MovieError, 0),
	}

	for id := from; ; id++ {
		if err := ctx.Err(); err != nil {
			return RangeResult{}, err
		}

		movie, err := s.storage.GetMovieByID(ctx, id)

		if errors.Is(err, domain.ErrMovieNotFound) {
			result.Errors = append(result.Errors, MovieError{
				ID:      id,
				Message: domain.ErrMovieNotFound.Error(),
				Cause:   domain.ErrMovieNotFound,
			})
		} else if err != nil {
			return RangeResult{}, fmt.Errorf(
				"не удалось получить фильм %d из PostgreSQL: %w",
				id,
				err,
			)
		} else if movie == nil {
			return RangeResult{}, fmt.Errorf(
				"хранилище вернуло пустой фильм для ID=%d",
				id,
			)
		} else {
			result.Movies = append(result.Movies, *movie)
		}

		if id == to {
			break
		}
	}

	return result, nil
}

// GetLastMovie возвращает фильм с максимальным ID из PostgreSQL.
func (s *Service) GetLastMovie(
	ctx context.Context,
) (*domain.Movie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	movie, err := s.storage.GetLastMovie(ctx)
	if err != nil {
		return nil, fmt.Errorf(
			"не удалось получить последний фильм из PostgreSQL: %w",
			err,
		)
	}

	return movie, nil
}

// GetMovies возвращает все фильмы из локального хранилища.
func (s *Service) GetMovies(
	ctx context.Context,
) ([]domain.Movie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return s.storage.GetMovies(ctx)
}
