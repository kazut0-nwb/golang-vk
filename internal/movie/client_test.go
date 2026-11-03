package movie

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchMovie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/7/info.0.json":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"title":"Тестовый фильм","year":2024,"director":"Тест"}`)
		case "/8/info.0.json":
			http.NotFound(w, r)
		default:
			http.Error(w, "ошибка стенда", http.StatusBadGateway)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL+"/", time.Second)

	got, err := client.FetchMovie(context.Background(), 7)
	if err != nil {
		t.Fatalf("ошибка загрузки: %v", err)
	}

	if got.ID != 7 || got.Title != "Тестовый фильм" || got.Year != 2024 {
		t.Fatalf("неожиданный фильм: %+v", got)
	}

	_, err = client.FetchMovie(context.Background(), 8)
	if !errors.Is(err, ErrMovieNotFound) {
		t.Fatalf("ожидали ErrMovieNotFound, получили: %v", err)
	}

	_, err = client.FetchMovie(context.Background(), 9)
	var upstreamErr *UpstreamStatusError

	if !errors.As(err, &upstreamErr) || upstreamErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("ожидали HTTP 502, получили: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.FetchMovie(ctx, 7)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ожидали отмену запроса, получили: %v", err)
	}
}
