package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	moviepb "my-go-app/api/proto"
	"my-go-app/internal/domain"
	"my-go-app/internal/handler"
	"my-go-app/internal/usecase"
)

// fakeMovieUseCase заменяет настоящий прикладной сервис.
type fakeMovieUseCase struct {
	loadCalled bool
	lastCalled bool
	from       int
	to         int
}

func (f *fakeMovieUseCase) LoadMovies(
	ctx context.Context,
	from, to int,
) (usecase.RangeResult, error) {
	f.loadCalled = true
	f.from = from
	f.to = to

	return usecase.RangeResult{
		Movies: []domain.Movie{
			{
				ID:       1,
				Title:    "Test Movie",
				Year:     2024,
				Director: "Test Director",
			},
		},
	}, nil
}

func (f *fakeMovieUseCase) GetLastMovie(
	ctx context.Context,
) (*domain.Movie, error) {
	f.lastCalled = true

	return &domain.Movie{
		ID:       42,
		Title:    "Last Movie",
		Year:     2025,
		Director: "Test Director",
	}, nil
}

// Проверяем, что fake соответствует интерфейсу.
var _ usecase.MovieUseCase = (*fakeMovieUseCase)(nil)

func TestHTTPHandlerWithFakeUseCase(t *testing.T) {
	fake := &fakeMovieUseCase{}
	h := handler.NewHTTPHandler(fake)

	req := httptest.NewRequest(
		http.MethodGet,
		"/load-movies?from=1&to=3",
		nil,
	)
	rec := httptest.NewRecorder()

	h.LoadMovies(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"ожидали HTTP 200, получили %d",
			rec.Code,
		)
	}

	if !fake.loadCalled {
		t.Fatal("метод LoadMovies не был вызван")
	}

	if fake.from != 1 || fake.to != 3 {
		t.Fatalf(
			"неверный диапазон: from=%d, to=%d",
			fake.from,
			fake.to,
		)
	}

	var movies []domain.Movie
	if err := json.Unmarshal(rec.Body.Bytes(), &movies); err != nil {
		t.Fatalf("ошибка декодирования JSON: %v", err)
	}

	if len(movies) != 1 || movies[0].ID != 1 {
		t.Fatalf("неожиданный ответ: %+v", movies)
	}
}

func TestGRPCHandlerWithFakeUseCase(t *testing.T) {
	fake := &fakeMovieUseCase{}
	server := handler.NewGRPCServer(fake)

	result, err := server.GetLastMovie(
		context.Background(),
		&moviepb.GetLastMovieRequest{},
	)

	if err != nil {
		t.Fatalf("неожиданная ошибка gRPC: %v", err)
	}

	if !fake.lastCalled {
		t.Fatal("метод GetLastMovie не был вызван")
	}

	if result.GetId() != 42 {
		t.Fatalf(
			"ожидали фильм ID=42, получили ID=%d",
			result.GetId(),
		)
	}
}
