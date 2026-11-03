package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"my-go-app/internal/domain"
)

// MovieProvider описывает зависимость от источника фильмов.
type MovieProvider interface {
	FetchMovie(ctx context.Context, id int) (*domain.Movie, error)
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
	provider MovieProvider
	workers  int
}

func NewService(provider MovieProvider) *Service {
	return &Service{
		provider: provider,
		workers:  10,
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

type fetchResult struct {
	id    int
	movie *domain.Movie
	err   error
}

func (s *Service) LoadMovies(
	ctx context.Context,
	from, to int,
) (RangeResult, error) {
	if err := ValidateRange(from, to); err != nil {
		return RangeResult{}, err
	}

	if err := ctx.Err(); err != nil {
		return RangeResult{}, err
	}

	jobs := make(chan int)
	results := make(chan fetchResult)
	var wg sync.WaitGroup

	workers := s.workers
	if count := to - from + 1; count < workers {
		workers = count
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return

				case id, ok := <-jobs:
					if !ok {
						return
					}

					m, err := s.provider.FetchMovie(ctx, id)

					select {
					case <-ctx.Done():
						return

					case results <- fetchResult{
						id:    id,
						movie: m,
						err:   err,
					}:
					}
				}
			}
		}()
	}

	go func() {
		defer close(jobs)

		for id := from; ; id++ {
			select {
			case <-ctx.Done():
				return
			case jobs <- id:
			}

			if id == to {
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	output := RangeResult{
		Movies: make([]domain.Movie, 0),
		Errors: make([]MovieError, 0),
	}

	for {
		select {
		case <-ctx.Done():
			return RangeResult{}, ctx.Err()

		case item, ok := <-results:
			if !ok {
				sort.Slice(output.Movies, func(i, j int) bool {
					return output.Movies[i].ID < output.Movies[j].ID
				})

				sort.Slice(output.Errors, func(i, j int) bool {
					return output.Errors[i].ID < output.Errors[j].ID
				})

				return output, nil
			}

			if item.err != nil {
				output.Errors = append(output.Errors, MovieError{
					ID:      item.id,
					Message: item.err.Error(),
					Cause:   item.err,
				})
			} else {
				output.Movies = append(output.Movies, *item.movie)
			}
		}
	}
}

func (s *Service) GetLastMovie(ctx context.Context) (*domain.Movie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	last, err := s.provider.FetchMovie(ctx, 1)
	if err != nil {
		return nil, err
	}

	left, right := 1, 2
	maxInt := int(^uint(0) >> 1)

	for {
		m, err := s.provider.FetchMovie(ctx, right)

		if errors.Is(err, domain.ErrMovieNotFound) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf(
				"не удалось найти верхнюю границу (id=%d): %w",
				right, err,
			)
		}

		last = m
		left = right

		if right > maxInt/2 {
			return nil, errors.New(
				"слишком большой идентификатор последнего фильма",
			)
		}

		right *= 2
	}

	for right-left > 1 {
		mid := left + (right-left)/2
		m, err := s.provider.FetchMovie(ctx, mid)

		if errors.Is(err, domain.ErrMovieNotFound) {
			right = mid
		} else if err != nil {
			return nil, fmt.Errorf(
				"не удалось проверить фильм %d: %w",
				mid, err,
			)
		} else {
			left = mid
			last = m
		}
	}

	return last, nil
}
