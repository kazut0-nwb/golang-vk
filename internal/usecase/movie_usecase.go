package usecase

import (
	"context"

	"my-go-app/internal/domain"
)

// MovieUseCase описывает операции приложения,
// доступные транспортному слою.
type MovieUseCase interface {
	LoadMovies(
		ctx context.Context,
		from, to int,
	) (RangeResult, error)

	GetLastMovie(
		ctx context.Context,
	) (*domain.Movie, error)
}

// Проверяем на этапе компиляции, что Service
// реализует интерфейс MovieUseCase.
var _ MovieUseCase = (*Service)(nil)
