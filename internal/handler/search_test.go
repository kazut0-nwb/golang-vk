package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"my-go-app/internal/domain"
	"my-go-app/internal/usecase"
)

type searchUseCaseStub struct {
	movies   []domain.Movie
	err      error
	gotQuery string
	called   bool
}

func (s *searchUseCaseStub) Search(
	ctx context.Context,
	query string,
) ([]domain.Movie, error) {
	s.called = true
	s.gotQuery = query

	return s.movies, s.err
}

func newSearchTestHandler(
	searchService usecase.MovieSearchUseCase,
) *HTTPHandler {
	return NewHTTPHandler(nil, searchService)
}

func TestSearchMoviesSuccess(t *testing.T) {
	expected := []domain.Movie{
		{
			ID:       1251,
			Title:    "Open Season",
			Year:     2006,
			Director: "Anthony Stacchi",
		},
	}

	service := &searchUseCaseStub{
		movies: expected,
	}

	h := newSearchTestHandler(service)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(
		http.MethodGet,
		"/search?q=%20Open%20",
		nil,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"ожидали HTTP 200, получили %d: %s",
			rec.Code,
			rec.Body.String(),
		)
	}

	if !strings.Contains(
		rec.Header().Get("Content-Type"),
		"application/json",
	) {
		t.Fatal("неверный Content-Type")
	}

	if !service.called {
		t.Fatal("поисковый сервис не был вызван")
	}

	if service.gotQuery != "Open" {
		t.Fatalf(
			"ожидали запрос Open, получили %q",
			service.gotQuery,
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"Open Season",
	) {
		t.Fatalf("ответ не содержит фильм: %s", rec.Body.String())
	}
}

func TestSearchMoviesEmptyQuery(t *testing.T) {
	service := &searchUseCaseStub{}

	h := newSearchTestHandler(service)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(
		http.MethodGet,
		"/search?q=%20%20%20",
		nil,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf(
			"ожидали HTTP 400, получили %d",
			rec.Code,
		)
	}

	if strings.TrimSpace(rec.Body.String()) == "" {
		t.Fatal("ожидалось описание ошибки")
	}

	if service.called {
		t.Fatal("поисковый сервис не должен вызываться")
	}
}

func TestSearchMoviesNoResults(t *testing.T) {
	service := &searchUseCaseStub{
		movies: []domain.Movie{},
	}

	h := newSearchTestHandler(service)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(
		http.MethodGet,
		"/search?q=Unknown",
		nil,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"ожидали HTTP 200, получили %d",
			rec.Code,
		)
	}

	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf(
			"ожидали [], получили %s",
			rec.Body.String(),
		)
	}
}

func TestSearchMoviesInternalError(t *testing.T) {
	service := &searchUseCaseStub{
		err: errors.New("Redis unavailable"),
	}

	h := newSearchTestHandler(service)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(
		http.MethodGet,
		"/search?q=Solaris",
		nil,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf(
			"ожидали HTTP 500, получили %d",
			rec.Code,
		)
	}
}
