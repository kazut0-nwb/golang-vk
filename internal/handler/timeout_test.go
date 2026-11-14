package handler_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"my-go-app/internal/domain"
	"my-go-app/internal/handler"
	"my-go-app/internal/usecase"
)

// Заглушка, возвращающая заранее подготовленный результат.
type rangeResultUseCase struct {
	result usecase.RangeResult
}

func (s *rangeResultUseCase) LoadMovies(
	ctx context.Context,
	from, to int,
) (usecase.RangeResult, error) {
	return s.result, nil
}

func (s *rangeResultUseCase) GetLastMovie(
	ctx context.Context,
) (*domain.Movie, error) {
	return nil, domain.ErrMovieNotFound
}

func TestHTTPHandlerTimeoutStatuses(t *testing.T) {
	tests := []struct {
		name   string
		result usecase.RangeResult
		status int
	}{
		{
			name: "все запросы завершились таймаутом",
			result: usecase.RangeResult{
				Errors: []usecase.MovieError{
					{
						ID: 1,
						Cause: fmt.Errorf(
							"ошибка клиента: %w",
							context.DeadlineExceeded,
						),
					},
				},
			},
			status: http.StatusGatewayTimeout,
		},
		{
			name: "частично успешная загрузка",
			result: usecase.RangeResult{
				Movies: []domain.Movie{
					{ID: 1, Title: "Movie 1"},
				},
				Errors: []usecase.MovieError{
					{
						ID:    2,
						Cause: context.DeadlineExceeded,
					},
				},
			},
			status: http.StatusMultiStatus,
		},
		{
			name: "все фильмы отсутствуют",
			result: usecase.RangeResult{
				Errors: []usecase.MovieError{
					{
						ID:    1,
						Cause: domain.ErrMovieNotFound,
					},
				},
			},
			status: http.StatusNotFound,
		},
		{
			name: "другая ошибка внешнего сервиса",
			result: usecase.RangeResult{
				Errors: []usecase.MovieError{
					{
						ID:    1,
						Cause: errors.New("upstream unavailable"),
					},
				},
			},
			status: http.StatusBadGateway,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &rangeResultUseCase{
				result: tt.result,
			}

			h := handler.NewHTTPHandler(fake)

			req := httptest.NewRequest(
				http.MethodGet,
				"/load-movies?from=1&to=2",
				nil,
			)
			rec := httptest.NewRecorder()

			h.LoadMovies(rec, req)

			if rec.Code != tt.status {
				t.Fatalf(
					"ожидали HTTP %d, получили %d: %s",
					tt.status,
					rec.Code,
					rec.Body.String(),
				)
			}
		})
	}
}
