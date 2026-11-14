package testutil

import (
	"context"

	"my-go-app/internal/domain"
)

type MovieStorage struct {
	Movies map[int]domain.Movie
}

func NewMovieStorage(movies ...domain.Movie) *MovieStorage {
	storage := &MovieStorage{
		Movies: make(map[int]domain.Movie, len(movies)),
	}

	for _, movie := range movies {
		storage.Movies[movie.ID] = movie
	}

	return storage
}

func (s *MovieStorage) SaveMovie(
	ctx context.Context,
	movie *domain.Movie,
) error {
	s.Movies[movie.ID] = *movie
	return nil
}

func (s *MovieStorage) GetMovieByID(
	ctx context.Context,
	id int,
) (*domain.Movie, error) {
	movie, ok := s.Movies[id]
	if !ok {
		return nil, domain.ErrMovieNotFound
	}

	return &movie, nil
}

func (s *MovieStorage) GetMovies(
	ctx context.Context,
) ([]domain.Movie, error) {
	movies := make([]domain.Movie, 0, len(s.Movies))

	for _, movie := range s.Movies {
		movies = append(movies, movie)
	}

	return movies, nil
}

func (s *MovieStorage) GetLastMovie(
	ctx context.Context,
) (*domain.Movie, error) {
	var last domain.Movie
	found := false

	for _, movie := range s.Movies {
		if !found || movie.ID > last.ID {
			last = movie
			found = true
		}
	}

	if !found {
		return nil, domain.ErrMovieNotFound
	}

	return &last, nil
}
