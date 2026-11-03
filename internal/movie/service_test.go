package movie

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestLoadMovies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/1/info.0.json":
			fmt.Fprint(w, `{"id":1,"title":"Movie 1"}`)
		case "/2/info.0.json":
			http.NotFound(w, r)
		case "/3/info.0.json":
			http.Error(w, "upstream failure", http.StatusBadGateway)
		default:
			http.Error(w, "unexpected", 500)
		}
	}))
	defer server.Close()

	service := NewService(NewClient(server.URL, time.Second))

	res, err := service.LoadMovies(context.Background(), 1, 3)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if len(res.Movies) != 1 || res.Movies[0].ID != 1 {
		t.Fatalf("неверные фильмы: %+v", res.Movies)
	}

	if len(res.Errors) != 2 || res.Errors[0].ID != 2 || res.Errors[1].ID != 3 {
		t.Fatalf("неверные ошибки: %+v", res.Errors)
	}

	if !errors.Is(res.Errors[0].Cause, ErrMovieNotFound) {
		t.Fatalf("ожидали 404, получили %v", res.Errors[0].Cause)
	}

	for _, r := range [][2]int{{0, 2}, {3, 2}, {-1, 2}} {
		_, err := service.LoadMovies(context.Background(), r[0], r[1])
		if !errors.Is(err, ErrInvalidRange) {
			t.Fatalf("диапазон %v должен быть ошибкой, получили %v", r, err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = service.LoadMovies(ctx, 1, 3)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ожидали отмену, получили %v", err)
	}
}

func TestGetLastMovie(t *testing.T) {
	var count atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)

		segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(segments) != 2 || segments[1] != "info.0.json" {
			http.Error(w, "unexpected path", 500)
			return
		}

		id, err := strconv.Atoi(segments[0])
		if err != nil {
			http.Error(w, "invalid id", 400)
			return
		}

		if id > 7 {
			http.NotFound(w, r)
			return
		}

		fmt.Fprintf(w, `{"id":%d,"title":"Movie"}`, id)
	}))
	defer server.Close()

	service := NewService(NewClient(server.URL, time.Second))

	m, err := service.GetLastMovie(context.Background())
	if err != nil {
		t.Fatalf("ошибка поиска: %v", err)
	}

	if m.ID != 7 {
		t.Fatalf("ожидали фильм 7, получили %+v", m)
	}

	if count.Load() > 8 {
		t.Fatalf("слишком много HTTP-запросов: %d", count.Load())
	}
}

func TestGetLastMovieUpstreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/1/info.0.json" {
			fmt.Fprint(w, `{"id":1}`)
			return
		}
		http.Error(w, "not working", http.StatusBadGateway)
	}))
	defer server.Close()

	service := NewService(NewClient(server.URL, time.Second))

	_, err := service.GetLastMovie(context.Background())

	var upstream *UpstreamStatusError
	if !errors.As(err, &upstream) || upstream.StatusCode != 502 {
		t.Fatalf("ожидали 502, получили %v", err)
	}
}
