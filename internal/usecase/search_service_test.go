package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"

	"my-go-app/internal/domain"
)

type searchTestIndex struct {
	ids    []int
	tokens []string
	calls  int
	err    error
}

func (f *searchTestIndex) SearchIDs(
	ctx context.Context,
	tokens []string,
) ([]int, error) {
	f.calls++
	f.tokens = append([]string(nil), tokens...)

	return f.ids, f.err
}

type searchTestStorage struct {
	movies []domain.Movie
	ids    []int
	calls  int
	err    error
}

func (f *searchTestStorage) GetMoviesByIDs(
	ctx context.Context,
	ids []int,
) ([]domain.Movie, error) {
	f.calls++
	f.ids = append([]int(nil), ids...)

	return f.movies, f.err
}

type searchTestCache struct {
	values map[string]string
	ttl    time.Duration
	gets   int
	sets   int
	err    error
}

func newSearchTestCache() *searchTestCache {
	return &searchTestCache{
		values: make(map[string]string),
	}
}

func (f *searchTestCache) Get(
	ctx context.Context,
	key string,
) (string, bool, error) {
	f.gets++

	if f.err != nil {
		return "", false, f.err
	}

	value, found := f.values[key]
	return value, found, nil
}

func (f *searchTestCache) Set(
	ctx context.Context,
	key string,
	value string,
	ttl time.Duration,
) error {
	f.sets++

	if f.err != nil {
		return f.err
	}

	f.values[key] = value
	f.ttl = ttl

	return nil
}

func TestSearchServiceSearch(t *testing.T) {
	expected := []domain.Movie{
		{
			ID:       1,
			Title:    "Solaris",
			Year:     1972,
			Director: "Andrei Tarkovsky",
		},
	}

	index := &searchTestIndex{
		ids: []int{1},
	}

	storage := &searchTestStorage{
		movies: expected,
	}

	cache := newSearchTestCache()

	service := NewSearchService(storage, index, cache)

	result, err := service.Search(
		context.Background(),
		" Solaris Tarkovsky ",
	)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf(
			"ожидали %+v, получили %+v",
			expected,
			result,
		)
	}

	expectedTokens := []string{"solaris", "tarkovsky"}

	if !reflect.DeepEqual(index.tokens, expectedTokens) {
		t.Fatalf(
			"ожидали токены %v, получили %v",
			expectedTokens,
			index.tokens,
		)
	}

	if storage.calls != 1 {
		t.Fatalf(
			"ожидали 1 обращение к PostgreSQL, получили %d",
			storage.calls,
		)
	}

	if cache.sets != 1 {
		t.Fatalf(
			"ожидали 1 запись в кэш, получили %d",
			cache.sets,
		)
	}

	if cache.ttl != time.Hour {
		t.Fatalf(
			"ожидали TTL 1 час, получили %v",
			cache.ttl,
		)
	}

	// Проверяем содержимое кэша.
	var cached []domain.Movie

	err = json.Unmarshal(
		[]byte(cache.values[makeCacheKey("Solaris Tarkovsky")]),
		&cached,
	)
	if err != nil {
		t.Fatalf("ошибка чтения тестового кэша: %v", err)
	}

	if !reflect.DeepEqual(cached, expected) {
		t.Fatalf("неверное содержимое кэша: %+v", cached)
	}
}

func TestSearchServiceCacheHit(t *testing.T) {
	expected := []domain.Movie{
		{
			ID:       1,
			Title:    "Solaris",
			Year:     1972,
			Director: "Andrei Tarkovsky",
		},
	}

	data, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}

	cache := newSearchTestCache()
	cache.values[makeCacheKey("Solaris")] = string(data)

	index := &searchTestIndex{}
	storage := &searchTestStorage{}

	service := NewSearchService(storage, index, cache)

	result, err := service.Search(
		context.Background(),
		"Solaris",
	)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("неверный результат: %+v", result)
	}

	if index.calls != 0 {
		t.Fatal("при попадании в кэш Redis индекс не должен вызываться")
	}

	if storage.calls != 0 {
		t.Fatal("при попадании в кэш PostgreSQL не должен вызываться")
	}

	if cache.sets != 0 {
		t.Fatal("TTL кэша не должен обновляться при чтении")
	}
}

func TestSearchServiceNoResults(t *testing.T) {
	index := &searchTestIndex{
		ids: []int{},
	}

	storage := &searchTestStorage{}
	cache := newSearchTestCache()

	service := NewSearchService(storage, index, cache)

	result, err := service.Search(
		context.Background(),
		"UnknownMovie",
	)
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}

	if result == nil {
		t.Fatal("ожидался пустой массив, получен nil")
	}

	if len(result) != 0 {
		t.Fatalf("ожидали 0 фильмов, получили %d", len(result))
	}

	if storage.calls != 0 {
		t.Fatal("PostgreSQL не должен вызываться без найденных ID")
	}

	if cache.sets != 1 {
		t.Fatal("пустой успешный результат должен кэшироваться")
	}

	if cache.values[makeCacheKey("UnknownMovie")] != "[]" {
		t.Fatalf(
			"ожидали кэшированный массив [], получили %q",
			cache.values[makeCacheKey("UnknownMovie")],
		)
	}
}

func TestSearchServiceEmptyQuery(t *testing.T) {
	service := NewSearchService(
		&searchTestStorage{},
		&searchTestIndex{},
		newSearchTestCache(),
	)

	_, err := service.Search(
		context.Background(),
		"   ",
	)

	if !errors.Is(err, ErrEmptySearchQuery) {
		t.Fatalf(
			"ожидали ErrEmptySearchQuery, получили %v",
			err,
		)
	}
}
