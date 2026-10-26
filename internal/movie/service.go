package movie

import (
	"context"

	"my-go-app/internal/domain"
	"my-go-app/internal/usecase"
)

var ErrInvalidRange = domain.ErrInvalidRange

type MovieError = usecase.MovieError

type RangeResult = usecase.RangeResult

type Service struct {
	inner *usecase.Service
}

func NewService(client *Client) *Service {
	return &Service{
		inner: usecase.NewService(client),
	}
}

func ValidateRange(from, to int) error {
	return usecase.ValidateRange(from, to)
}

func (s *Service) LoadMovies(
	ctx context.Context,
	from, to int,
) (RangeResult, error) {
	return s.inner.LoadMovies(ctx, from, to)
}

func (s *Service) GetLastMovie(
	ctx context.Context,
) (*Movie, error) {
	return s.inner.GetLastMovie(ctx)
}
