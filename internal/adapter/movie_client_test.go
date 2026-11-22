package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"my-go-app/internal/domain"
)

// Проверяем успешное получение фильма.
func TestFetchMovieSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_ = json.NewEncoder(w).Encode(domain.Movie{
				ID:       1,
				Title:    "Solaris",
				Year:     1972,
				Director: "Andrei Tarkovsky",
			})
		},
	))
	defer server.Close()

	client := NewClient(server.URL, time.Second)

	movie, err := client.FetchMovie(
		context.Background(),
		1,
	)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if movie.ID != 1 {
		t.Errorf("ожидали ID=1, получили %d", movie.ID)
	}

	if movie.Title != "Solaris" {
		t.Errorf("неверное название: %s", movie.Title)
	}
}

// Проверяем повторные попытки при HTTP 503.
func TestFetchMovieRetriesTemporaryStatus(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			attempt := calls.Add(1)

			if attempt < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}

			w.Header().Set(
				"Content-Type",
				"application/json",
			)

			_, _ = w.Write([]byte(
				`{"id":1,"title":"Solaris","year":1972,"director":"Andrei Tarkovsky"}`,
			))
		},
	))
	defer server.Close()

	client := NewClient(server.URL, time.Second)

	movie, err := client.FetchMovie(
		context.Background(),
		1,
	)
	if err != nil {
		t.Fatalf("ожидали успешный запрос: %v", err)
	}

	if movie.ID != 1 {
		t.Fatalf("неверный ID фильма: %d", movie.ID)
	}

	if calls.Load() != 3 {
		t.Fatalf(
			"ожидали 3 попытки, получили %d",
			calls.Load(),
		)
	}
}

// Проверяем, что HTTP 404 не вызывает повторных запросов.
func TestFetchMovieNotFoundDoesNotRetry(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusNotFound)
		},
	))
	defer server.Close()

	client := NewClient(server.URL, time.Second)

	_, err := client.FetchMovie(
		context.Background(),
		100,
	)

	if !errors.Is(err, domain.ErrMovieNotFound) {
		t.Fatalf(
			"ожидали ErrMovieNotFound, получили %v",
			err,
		)
	}

	if calls.Load() != 1 {
		t.Fatalf(
			"404 не должен повторяться, запросов: %d",
			calls.Load(),
		)
	}
}

// Проверяем, что обычный HTTP 400 не повторяется.
func TestFetchMovieBadRequestDoesNotRetry(t *testing.T) {
	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			w.WriteHeader(http.StatusBadRequest)
		},
	))
	defer server.Close()

	client := NewClient(server.URL, time.Second)

	_, err := client.FetchMovie(
		context.Background(),
		1,
	)

	var statusErr *UpstreamStatusError

	if !errors.As(err, &statusErr) {
		t.Fatalf(
			"ожидали UpstreamStatusError, получили %v",
			err,
		)
	}

	if statusErr.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"ожидали HTTP 400, получили %d",
			statusErr.StatusCode,
		)
	}

	if calls.Load() != 1 {
		t.Fatalf(
			"HTTP 400 не должен повторяться, запросов: %d",
			calls.Load(),
		)
	}
}

// Проверяем отмену контекста во время ожидания повтора.
func TestFetchMovieCancellationDuringRetry(t *testing.T) {
	var calls atomic.Int32

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			cancel()

			w.WriteHeader(http.StatusServiceUnavailable)
		},
	))
	defer server.Close()

	client := NewClient(server.URL, time.Second)

	_, err := client.FetchMovie(ctx, 1)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"ожидали context.Canceled, получили %v",
			err,
		)
	}

	if calls.Load() != 1 {
		t.Fatalf(
			"после отмены контекста запрос повторился: %d",
			calls.Load(),
		)
	}
}
