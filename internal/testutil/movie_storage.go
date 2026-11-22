package testutil

import (
	"context"
	"sync"

	"my-go-app/internal/domain"
)

type MovieStorage struct {
	mu     sync.RWMutex
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
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Movies[movie.ID] = *movie

	return nil
}

func (s *MovieStorage) GetMovieByID(
	ctx context.Context,
	id int,
) (*domain.Movie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	movie, ok := s.Movies[id]
	if !ok {
		return nil, domain.ErrMovieNotFound
	}

	return &movie, nil
}

func (s *MovieStorage) GetMovies(
	ctx context.Context,
) ([]domain.Movie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	movies := make([]domain.Movie, 0, len(s.Movies))

	for _, movie := range s.Movies {
		movies = append(movies, movie)
	}

	return movies, nil
}

func (s *MovieStorage) GetLastMovie(
	ctx context.Context,
) (*domain.Movie, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
