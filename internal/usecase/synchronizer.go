package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"my-go-app/internal/domain"
)

const defaultSyncWorkers = 5

// MovieSynchronizer синхронизирует фильмы
// с внешнего стенда в локальную базу данных.
type MovieSynchronizer struct {
	provider MovieProvider
	storage  MovieStorage
	workers  int
}

func NewMovieSynchronizer(
	provider MovieProvider,
	storage MovieStorage,
) *MovieSynchronizer {
	return NewMovieSynchronizerWithWorkers(
		provider,
		storage,
		defaultSyncWorkers,
	)
}

// NewMovieSynchronizerWithWorkers создаёт синхронизатор
// с заданным количеством параллельных workers.
func NewMovieSynchronizerWithWorkers(
	provider MovieProvider,
	storage MovieStorage,
	workers int,
) *MovieSynchronizer {
	if workers <= 0 {
		workers = defaultSyncWorkers
	}

	return &MovieSynchronizer{
		provider: provider,
		storage:  storage,
		workers:  workers,
	}
}

// Sync синхронизирует отсутствующие фильмы
// с использованием пула workers.
func (s *MovieSynchronizer) Sync(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Определяем последний доступный фильм на стенде.
	lastMovie, err := s.findLastMovie(ctx)
	if err != nil {
		return fmt.Errorf(
			"не удалось определить последний фильм: %w",
			err,
		)
	}

	// Получаем фильмы, которые уже находятся в PostgreSQL.
	existingMovies, err := s.storage.GetMovies(ctx)
	if err != nil {
		return fmt.Errorf(
			"не удалось получить фильмы из БД: %w",
			err,
		)
	}

	existingIDs := make(map[int]struct{}, len(existingMovies))

	for _, movie := range existingMovies {
		existingIDs[movie.ID] = struct{}{}
	}

	log.Printf(
		"синхронизация: последний ID на стенде — %d, уже сохранено фильмов — %d",
		lastMovie.ID,
		len(existingMovies),
	)

	return s.syncMissingMovies(
		ctx,
		lastMovie.ID,
		existingIDs,
	)
}

// syncMissingMovies загружает отсутствующие фильмы
// с помощью ограниченного пула workers.
func (s *MovieSynchronizer) syncMissingMovies(
	ctx context.Context,
	lastID int,
	existingIDs map[int]struct{},
) error {
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Считаем количество ID, которые необходимо проверить.
	total := 0

	for id := 1; ; id++ {
		if _, exists := existingIDs[id]; !exists {
			total++
		}

		if id == lastID {
			break
		}
	}

	log.Printf(
		"синхронизация: запланировано проверить %d ID, workers: %d",
		total,
		s.workers,
	)

	jobs := make(chan int, s.workers)

	var wg sync.WaitGroup
	var errOnce sync.Once
	var syncErr error

	var processed int64
	var saved int64
	var notFound int64

	// Запускаем workers.
	for workerID := 0; workerID < s.workers; workerID++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-workCtx.Done():
					return

				case id, ok := <-jobs:
					if !ok {
						return
					}

					wasSaved, err := s.syncMovie(workCtx, id)

					if err != nil {
						if workCtx.Err() != nil {
							return
						}

						errOnce.Do(func() {
							syncErr = fmt.Errorf(
								"ошибка обработки фильма %d: %w",
								id,
								err,
							)

							cancel()
						})

						return
					}

					if wasSaved {
						atomic.AddInt64(&saved, 1)
					} else {
						atomic.AddInt64(&notFound, 1)
					}

					done := atomic.AddInt64(&processed, 1)

					if done%100 == 0 || done == int64(total) {
						log.Printf(
							"синхронизация: обработано %d/%d ID, сохранено: %d, отсутствует: %d",
							done,
							total,
							atomic.LoadInt64(&saved),
							atomic.LoadInt64(&notFound),
						)
					}
				}
			}
		}()
	}

	// Передаём workers только отсутствующие ID.
feedJobs:
	for id := 1; ; id++ {
		if ctx.Err() != nil {
			break
		}

		if _, exists := existingIDs[id]; !exists {
			select {
			case <-workCtx.Done():
				break feedJobs

			case jobs <- id:
			}
		}

		if id == lastID {
			break
		}
	}

	close(jobs)

	wg.Wait()

	log.Printf(
		"синхронизация завершена: обработано %d/%d ID, сохранено: %d, отсутствует: %d",
		atomic.LoadInt64(&processed),
		total,
		atomic.LoadInt64(&saved),
		atomic.LoadInt64(&notFound),
	)

	if syncErr != nil {
		return syncErr
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

// syncMovie загружает и сохраняет один фильм.
func (s *MovieSynchronizer) syncMovie(
	ctx context.Context,
	id int,
) (bool, error) {
	movie, err := s.fetchMovie(ctx, id)

	if errors.Is(err, domain.ErrMovieNotFound) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf(
			"не удалось получить фильм: %w",
			err,
		)
	}

	if err := s.storage.SaveMovie(ctx, movie); err != nil {
		return false, fmt.Errorf(
			"не удалось сохранить фильм %d: %w",
			id,
			err,
		)
	}

	return true, nil
}

// fetchMovie получает фильм и проверяет корректность его ID.
func (s *MovieSynchronizer) fetchMovie(
	ctx context.Context,
	id int,
) (*domain.Movie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	movie, err := s.provider.FetchMovie(ctx, id)
	if err != nil {
		return nil, err
	}

	if movie == nil {
		return nil, fmt.Errorf(
			"внешний клиент вернул пустой фильм %d",
			id,
		)
	}

	if movie.ID == 0 {
		movie.ID = id
	}

	if movie.ID != id {
		return nil, fmt.Errorf(
			"ID полученного фильма %d не совпадает с запрошенным %d",
			movie.ID,
			id,
		)
	}

	return movie, nil
}

// findLastMovie определяет последний доступный фильм.
// Используется экспоненциальный поиск
// с последующим бинарным поиском.
func (s *MovieSynchronizer) findLastMovie(
	ctx context.Context,
) (*domain.Movie, error) {
	fetch := func(id int) (*domain.Movie, error) {
		return s.fetchMovie(ctx, id)
	}

	last, err := fetch(1)
	if err != nil {
		return nil, fmt.Errorf(
			"не удалось получить первый фильм: %w",
			err,
		)
	}

	left, right := 1, 2
	maxInt := int(^uint(0) >> 1)

	// Экспоненциальный поиск верхней границы.
	for {
		movie, err := fetch(right)

		if errors.Is(err, domain.ErrMovieNotFound) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf(
				"ошибка проверки фильма %d: %w",
				right,
				err,
			)
		}

		last = movie
		left = right

		if right > maxInt/2 {
			return nil, errors.New(
				"слишком большой идентификатор последнего фильма",
			)
		}

		right *= 2
	}

	// Бинарный поиск последнего существующего фильма.
	for right-left > 1 {
		mid := left + (right-left)/2

		movie, err := fetch(mid)

		if errors.Is(err, domain.ErrMovieNotFound) {
			right = mid
			continue
		}

		if err != nil {
			return nil, fmt.Errorf(
				"ошибка проверки фильма %d: %w",
				mid,
				err,
			)
		}

		last = movie
		left = mid
	}

	return last, nil
}
