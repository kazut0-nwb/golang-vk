package postgres

import (
	"context"
	"fmt"

	"my-go-app/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MovieRepository struct {
	pool *pgxpool.Pool
}

func NewMovieRepository(pool *pgxpool.Pool) *MovieRepository {
	return &MovieRepository{
		pool: pool,
	}
}

// SaveMovie сохраняет фильм или обновляет существующий.
func (r *MovieRepository) SaveMovie(
	ctx context.Context,
	movie *domain.Movie,
) error {
	query := `
		INSERT INTO movies (id, title, year, director)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			year = EXCLUDED.year,
			director = EXCLUDED.director
	`

	_, err := r.pool.Exec(
		ctx,
		query,
		movie.ID,
		movie.Title,
		movie.Year,
		movie.Director,
	)

	if err != nil {
		return fmt.Errorf("не удалось сохранить фильм %d: %w", movie.ID, err)
	}

	return nil
}

// GetMovieByID возвращает фильм по ID.
func (r *MovieRepository) GetMovieByID(
	ctx context.Context,
	id int,
) (*domain.Movie, error) {
	query := `
		SELECT id, title, year, director
		FROM movies
		WHERE id = $1
	`

	var movie domain.Movie

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&movie.ID,
		&movie.Title,
		&movie.Year,
		&movie.Director,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrMovieNotFound
		}

		return nil, fmt.Errorf("не удалось получить фильм %d: %w", id, err)
	}

	return &movie, nil
}

// GetLastMovie возвращает фильм с максимальным ID.
func (r *MovieRepository) GetLastMovie(
	ctx context.Context,
) (*domain.Movie, error) {
	query := `
		SELECT id, title, year, director
		FROM movies
		ORDER BY id DESC
		LIMIT 1
	`

	var movie domain.Movie

	err := r.pool.QueryRow(ctx, query).Scan(
		&movie.ID,
		&movie.Title,
		&movie.Year,
		&movie.Director,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, domain.ErrMovieNotFound
		}

		return nil, fmt.Errorf("не удалось получить последний фильм: %w", err)
	}

	return &movie, nil
}

// GetMovies возвращает все фильмы из базы.
func (r *MovieRepository) GetMovies(
	ctx context.Context,
) ([]domain.Movie, error) {
	query := `
		SELECT id, title, year, director
		FROM movies
		ORDER BY id
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить фильмы: %w", err)
	}
	defer rows.Close()

	movies := make([]domain.Movie, 0)

	for rows.Next() {
		var movie domain.Movie

		if err := rows.Scan(
			&movie.ID,
			&movie.Title,
			&movie.Year,
			&movie.Director,
		); err != nil {
			return nil, fmt.Errorf("ошибка чтения фильма: %w", err)
		}

		movies = append(movies, movie)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка обхода результатов: %w", err)
	}

	return movies, nil
}
