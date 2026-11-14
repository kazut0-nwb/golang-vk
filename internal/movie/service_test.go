package movie

import (
	"context"
	"errors"
	"testing"

	"my-go-app/internal/domain"
	"my-go-app/internal/testutil"
)

func TestLoadMovies(t *testing.T) {
	storage := testutil.NewMovieStorage(
		domain.Movie{ID: 1, Title: "Movie 1"},
	)

	service := NewService(nil, storage)

	res, err := service.LoadMovies(context.Background(), 1, 3)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if len(res.Movies) != 1 || res.Movies[0].ID != 1 {
		t.Fatalf("неверные фильмы: %+v", res.Movies)
	}

	if len(res.Errors) != 2 ||
		res.Errors[0].ID != 2 ||
		res.Errors[1].ID != 3 {
		t.Fatalf("неверные ошибки: %+v", res.Errors)
	}

	if !errors.Is(res.Errors[0].Cause, ErrMovieNotFound) {
		t.Fatalf("ожидали ErrMovieNotFound, получили %v",
			res.Errors[0].Cause)
	}

	for _, r := range [][2]int{{0, 2}, {3, 2}, {-1, 2}} {
		_, err := service.LoadMovies(
			context.Background(),
			r[0],
			r[1],
		)

		if !errors.Is(err, ErrInvalidRange) {
			t.Fatalf("диапазон %v должен быть ошибкой, получили %v",
				r, err)
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
	storage := testutil.NewMovieStorage(
		domain.Movie{ID: 1, Title: "Movie 1"},
		domain.Movie{ID: 2, Title: "Movie 2"},
		domain.Movie{ID: 3, Title: "Movie 3"},
		domain.Movie{ID: 4, Title: "Movie 4"},
		domain.Movie{ID: 5, Title: "Movie 5"},
		domain.Movie{ID: 6, Title: "Movie 6"},
		domain.Movie{ID: 7, Title: "Movie 7"},
	)

	service := NewService(nil, storage)

	movie, err := service.GetLastMovie(context.Background())
	if err != nil {
		t.Fatalf("ошибка получения последнего фильма: %v", err)
	}

	if movie.ID != 7 {
		t.Fatalf("ожидали фильм 7, получили %+v", movie)
	}
}

type failingMovieStorage struct {
	*testutil.MovieStorage
	err error
}

func (s *failingMovieStorage) GetLastMovie(
	ctx context.Context,
) (*domain.Movie, error) {
	return nil, s.err
}

func TestGetLastMovieStorageError(t *testing.T) {
	storageError := errors.New("database unavailable")

	storage := &failingMovieStorage{
		MovieStorage: testutil.NewMovieStorage(),
		err:          storageError,
	}

	service := NewService(nil, storage)

	_, err := service.GetLastMovie(context.Background())

	if !errors.Is(err, storageError) {
		t.Fatalf("ожидали ошибку PostgreSQL, получили %v", err)
	}
}
