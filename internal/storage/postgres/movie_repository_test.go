package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"my-go-app/internal/domain"
)

func setupTestRepository(t *testing.T) *MovieRepository {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL не задан")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("не удалось подключиться к PostgreSQL: %v", err)
	}

	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `
		TRUNCATE TABLE movies
	`)
	if err != nil {
		t.Fatalf("не удалось очистить таблицу movies: %v", err)
	}

	return NewMovieRepository(pool)
}

func TestSaveAndGetMovie(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	expected := &domain.Movie{
		ID:       1,
		Title:    "To Rome with Love",
		Year:     2012,
		Director: "Woody Allen",
	}

	if err := repo.SaveMovie(ctx, expected); err != nil {
		t.Fatalf("ошибка сохранения фильма: %v", err)
	}

	actual, err := repo.GetMovieByID(ctx, expected.ID)
	if err != nil {
		t.Fatalf("ошибка получения фильма: %v", err)
	}

	if *actual != *expected {
		t.Errorf("ожидался фильм %+v, получен %+v", expected, actual)
	}
}

func TestSaveMovieDoesNotCreateDuplicates(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	movie := &domain.Movie{
		ID:       1,
		Title:    "Godzilla",
		Year:     1998,
		Director: "Roland Emmerich",
	}

	if err := repo.SaveMovie(ctx, movie); err != nil {
		t.Fatalf("ошибка первого сохранения: %v", err)
	}

	updated := &domain.Movie{
		ID:       1,
		Title:    "Godzilla Updated",
		Year:     1999,
		Director: "Roland Emmerich",
	}

	if err := repo.SaveMovie(ctx, updated); err != nil {
		t.Fatalf("ошибка повторного сохранения: %v", err)
	}

	movies, err := repo.GetMovies(ctx)
	if err != nil {
		t.Fatalf("ошибка получения списка фильмов: %v", err)
	}

	if len(movies) != 1 {
		t.Fatalf("ожидался один фильм, получено: %d", len(movies))
	}

	if movies[0].Title != updated.Title {
		t.Errorf("название не обновилось: %s", movies[0].Title)
	}
}

func TestGetLastMovie(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	movies := []domain.Movie{
		{ID: 1, Title: "Movie 1", Year: 2000, Director: "Director 1"},
		{ID: 2, Title: "Movie 2", Year: 2001, Director: "Director 2"},
		{ID: 3, Title: "Movie 3", Year: 2002, Director: "Director 3"},
	}

	for i := range movies {
		if err := repo.SaveMovie(ctx, &movies[i]); err != nil {
			t.Fatalf("ошибка сохранения фильма: %v", err)
		}
	}

	last, err := repo.GetLastMovie(ctx)
	if err != nil {
		t.Fatalf("ошибка получения последнего фильма: %v", err)
	}

	if last.ID != 3 {
		t.Errorf("ожидался ID 3, получен %d", last.ID)
	}
}

func TestGetMovieByIDNotFound(t *testing.T) {
	repo := setupTestRepository(t)
	ctx := context.Background()

	_, err := repo.GetMovieByID(ctx, 999)

	if err != domain.ErrMovieNotFound {
		t.Errorf("ожидалась ErrMovieNotFound, получена %v", err)
	}
}
