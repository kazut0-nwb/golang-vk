package grpcapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	moviepb "my-go-app/api/proto"
	"my-go-app/internal/movie"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	upstream := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/1/info.0.json":
				fmt.Fprint(w, `{
					"id": 1,
					"title": "Movie 1",
					"year": 2024,
					"director": "Director 1"
				}`)

			case "/2/info.0.json":
				http.NotFound(w, r)

			default:
				http.NotFound(w, r)
			}
		},
	))

	t.Cleanup(upstream.Close)

	client := movie.NewClient(upstream.URL, time.Second)
	service := movie.NewService(client)

	return NewServer(service)
}

func TestLoadMoviesPartialResult(t *testing.T) {
	server := newTestServer(t)

	result, err := server.LoadMovies(
		context.Background(),
		&moviepb.LoadMoviesRequest{
			From: 1,
			To:   2,
		},
	)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if len(result.GetMovies()) != 1 {
		t.Fatalf("ожидали один фильм, получили %d", len(result.GetMovies()))
	}

	if result.GetMovies()[0].GetId() != 1 {
		t.Fatalf("ожидали фильм с ID=1")
	}

	if len(result.GetErrors()) != 1 {
		t.Fatalf("ожидали одну ошибку, получили %d", len(result.GetErrors()))
	}

	if result.GetErrors()[0].GetId() != 2 {
		t.Fatalf("ожидали ошибку фильма с ID=2")
	}
}

func TestLoadMoviesInvalidRange(t *testing.T) {
	server := newTestServer(t)

	_, err := server.LoadMovies(
		context.Background(),
		&moviepb.LoadMoviesRequest{
			From: 5,
			To:   2,
		},
	)

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf(
			"ожидали InvalidArgument, получили %v",
			status.Code(err),
		)
	}
}

func TestGetLastMovie(t *testing.T) {
	server := newTestServer(t)

	result, err := server.GetLastMovie(
		context.Background(),
		&moviepb.GetLastMovieRequest{},
	)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if result.GetId() != 1 {
		t.Fatalf("ожидали фильм ID=1, получили %d", result.GetId())
	}

	if result.GetTitle() != "Movie 1" {
		t.Fatalf("неверное название фильма: %s", result.GetTitle())
	}
}

func TestGetLastMovieCanceled(t *testing.T) {
	server := newTestServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := server.GetLastMovie(
		ctx,
		&moviepb.GetLastMovieRequest{},
	)

	if status.Code(err) != codes.Canceled {
		t.Fatalf("ожидали Canceled, получили %v", status.Code(err))
	}
}
