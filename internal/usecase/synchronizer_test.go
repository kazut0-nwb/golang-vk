package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"my-go-app/internal/domain"
	"my-go-app/internal/testutil"
)

// Тестовая реализация внешнего источника фильмов.
type synchronizerTestProvider struct {
	movies      map[int]domain.Movie
	fetchErrors map[int]error
	calls       map[int]int
}

func newSynchronizerTestProvider(
	movies ...domain.Movie,
) *synchronizerTestProvider {
	provider := &synchronizerTestProvider{
		movies:      make(map[int]domain.Movie),
		fetchErrors: make(map[int]error),
		calls:       make(map[int]int),
	}

	for _, movie := range movies {
		provider.movies[movie.ID] = movie
	}

	return provider
}

func (p *synchronizerTestProvider) FetchMovie(
	ctx context.Context,
	id int,
) (*domain.Movie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	p.calls[id]++

	if err, ok := p.fetchErrors[id]; ok {
		return nil, err
	}

	movie, ok := p.movies[id]
	if !ok {
		return nil, domain.ErrMovieNotFound
	}

	return &movie, nil
}

// Тестовая реализация хранилища с возможностью
// имитировать ошибки PostgreSQL.
type synchronizerTestStorage struct {
	*testutil.MovieStorage

	getMoviesErr error
	saveErr      error
	saveCalls    int
}

func newSynchronizerTestStorage(
	movies ...domain.Movie,
) *synchronizerTestStorage {
	return &synchronizerTestStorage{
		MovieStorage: testutil.NewMovieStorage(movies...),
	}
}

func (s *synchronizerTestStorage) GetMovies(
	ctx context.Context,
) ([]domain.Movie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if s.getMoviesErr != nil {
		return nil, s.getMoviesErr
	}

	return s.MovieStorage.GetMovies(ctx)
}

func (s *synchronizerTestStorage) SaveMovie(
	ctx context.Context,
	movie *domain.Movie,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.saveCalls++

	if s.saveErr != nil {
		return s.saveErr
	}

	return s.MovieStorage.SaveMovie(ctx, movie)
}

func testMovie(id int) domain.Movie {
	return domain.Movie{
		ID:       id,
		Title:    fmt.Sprintf("Movie %d", id),
		Year:     2024,
		Director: "Test Director",
	}
}

// Проверяем загрузку отсутствующих фильмов.
func TestMovieSynchronizerSyncSavesMissingMovies(t *testing.T) {
	provider := newSynchronizerTestProvider(
		testMovie(1),
		testMovie(2),
		testMovie(3),
		testMovie(4),
	)

	storage := newSynchronizerTestStorage(
		testMovie(1),
		testMovie(3),
	)

	synchronizer := NewMovieSynchronizer(provider, storage)

	err := synchronizer.Sync(context.Background())
	if err != nil {
		t.Fatalf("неожиданная ошибка синхронизации: %v", err)
	}

	if len(storage.Movies) != 4 {
		t.Fatalf(
			"ожидали 4 фильма в хранилище, получили %d",
			len(storage.Movies),
		)
	}

	if storage.saveCalls != 2 {
		t.Fatalf(
			"ожидали 2 сохранения, получили %d",
			storage.saveCalls,
		)
	}

	if storage.Movies[2].Title != "Movie 2" {
		t.Fatal("фильм 2 не был сохранён корректно")
	}

	if storage.Movies[4].Title != "Movie 4" {
		t.Fatal("фильм 4 не был сохранён корректно")
	}

	if provider.calls[3] != 0 {
		t.Fatal("уже существующий фильм 3 повторно запрашивался")
	}
}

// Проверяем обработку отсутствующего ID.
func TestMovieSynchronizerSyncSkipsNotFoundMovies(t *testing.T) {
	provider := newSynchronizerTestProvider(
		testMovie(1),
		testMovie(2),
		testMovie(4),
	)

	storage := newSynchronizerTestStorage()

	synchronizer := NewMovieSynchronizer(provider, storage)

	err := synchronizer.Sync(context.Background())
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if len(storage.Movies) != 3 {
		t.Fatalf(
			"ожидали 3 фильма, получили %d",
			len(storage.Movies),
		)
	}

	if _, exists := storage.Movies[3]; exists {
		t.Fatal("несуществующий фильм 3 был сохранён")
	}

	if provider.calls[3] != 1 {
		t.Fatalf(
			"ожидали один запрос фильма 3, получили %d",
			provider.calls[3],
		)
	}
}

// Проверяем отсутствие дубликатов при повторной синхронизации.
func TestMovieSynchronizerSyncIsIdempotent(t *testing.T) {
	provider := newSynchronizerTestProvider(
		testMovie(1),
		testMovie(2),
		testMovie(3),
	)

	storage := newSynchronizerTestStorage()

	synchronizer := NewMovieSynchronizer(provider, storage)

	if err := synchronizer.Sync(context.Background()); err != nil {
		t.Fatalf("первая синхронизация завершилась ошибкой: %v", err)
	}

	if storage.saveCalls != 3 {
		t.Fatalf("ожидали 3 сохранения, получили %d", storage.saveCalls)
	}

	if err := synchronizer.Sync(context.Background()); err != nil {
		t.Fatalf("повторная синхронизация завершилась ошибкой: %v", err)
	}

	if len(storage.Movies) != 3 {
		t.Fatalf("появились дубликаты: %d фильмов", len(storage.Movies))
	}

	if storage.saveCalls != 3 {
		t.Fatalf(
			"повторная синхронизация вызвала дополнительные сохранения: %d",
			storage.saveCalls,
		)
	}
}

// Проверяем ошибку внешнего источника при поиске последнего фильма.
func TestMovieSynchronizerSyncReturnsProviderError(t *testing.T) {
	provider := newSynchronizerTestProvider(
		testMovie(1),
		testMovie(2),
		testMovie(3),
	)

	upstreamErr := errors.New("upstream unavailable")
	provider.fetchErrors[4] = upstreamErr

	storage := newSynchronizerTestStorage()

	synchronizer := NewMovieSynchronizer(provider, storage)

	err := synchronizer.Sync(context.Background())

	if !errors.Is(err, upstreamErr) {
		t.Fatalf("ожидали ошибку внешнего источника, получили %v", err)
	}
}

// Проверяем ошибку внешнего источника во время загрузки фильмов.
func TestMovieSynchronizerSyncReturnsFetchErrorDuringDownload(
	t *testing.T,
) {
	provider := newSynchronizerTestProvider(
		testMovie(1),
		testMovie(2),
		testMovie(4),
	)

	upstreamErr := errors.New("download failed")
	provider.fetchErrors[3] = upstreamErr

	storage := newSynchronizerTestStorage()

	synchronizer := NewMovieSynchronizer(provider, storage)

	err := synchronizer.Sync(context.Background())

	if !errors.Is(err, upstreamErr) {
		t.Fatalf("ожидали ошибку загрузки, получили %v", err)
	}
}

// Проверяем ошибку чтения из PostgreSQL.
func TestMovieSynchronizerSyncReturnsStorageReadError(t *testing.T) {
	provider := newSynchronizerTestProvider(
		testMovie(1),
		testMovie(2),
		testMovie(3),
	)

	storageErr := errors.New("database unavailable")

	storage := newSynchronizerTestStorage()
	storage.getMoviesErr = storageErr

	synchronizer := NewMovieSynchronizer(provider, storage)

	err := synchronizer.Sync(context.Background())

	if !errors.Is(err, storageErr) {
		t.Fatalf("ожидали ошибку PostgreSQL, получили %v", err)
	}
}

// Проверяем ошибку сохранения в PostgreSQL.
func TestMovieSynchronizerSyncReturnsStorageSaveError(t *testing.T) {
	provider := newSynchronizerTestProvider(
		testMovie(1),
		testMovie(2),
		testMovie(3),
	)

	storageErr := errors.New("insert failed")

	storage := newSynchronizerTestStorage()
	storage.saveErr = storageErr

	synchronizer := NewMovieSynchronizer(provider, storage)

	err := synchronizer.Sync(context.Background())

	if !errors.Is(err, storageErr) {
		t.Fatalf("ожидали ошибку сохранения, получили %v", err)
	}
}

// Проверяем отмену контекста.
func TestMovieSynchronizerSyncCanceled(t *testing.T) {
	provider := newSynchronizerTestProvider(testMovie(1))
	storage := newSynchronizerTestStorage()

	synchronizer := NewMovieSynchronizer(provider, storage)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := synchronizer.Sync(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ожидали context.Canceled, получили %v", err)
	}

	if len(provider.calls) != 0 {
		t.Fatal("внешний источник вызывался после отмены контекста")
	}
}

// Проверяем ситуацию, когда на стенде нет первого фильма.
func TestMovieSynchronizerSyncReturnsErrorWhenNoMoviesExist(
	t *testing.T,
) {
	provider := newSynchronizerTestProvider()
	storage := newSynchronizerTestStorage()

	synchronizer := NewMovieSynchronizer(provider, storage)

	err := synchronizer.Sync(context.Background())

	if !errors.Is(err, domain.ErrMovieNotFound) {
		t.Fatalf("ожидали ErrMovieNotFound, получили %v", err)
	}
}

// Проверяем защиту от некорректного ID в ответе стенда.
func TestMovieSynchronizerSyncRejectsMismatchedID(t *testing.T) {
	provider := newSynchronizerTestProvider(
		testMovie(1),
	)

	provider.movies[1] = domain.Movie{
		ID:    99,
		Title: "Wrong movie",
	}

	storage := newSynchronizerTestStorage()

	synchronizer := NewMovieSynchronizer(provider, storage)

	err := synchronizer.Sync(context.Background())

	if err == nil {
		t.Fatal("ожидали ошибку несовпадения ID")
	}

	if len(storage.Movies) != 0 {
		t.Fatal("фильм с некорректным ID был сохранён")
	}
}
