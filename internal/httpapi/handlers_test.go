package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"my-go-app/internal/domain"
	"my-go-app/internal/movie"
	"my-go-app/internal/testutil"
)

func newTestMux(t *testing.T) *http.ServeMux {
	t.Helper()

	storage := testutil.NewMovieStorage(
		domain.Movie{ID: 1, Title: "Movie 1"},
		domain.Movie{ID: 2, Title: "Movie 2"},
	)

	service := movie.NewService(nil, storage)

	mux := http.NewServeMux()
	NewHandler(service).RegisterRoutes(mux)

	return mux
}

func TestLoadMoviesHandler(t *testing.T) {
	mux := newTestMux(t)

	tests := []struct {
		name   string
		url    string
		status int
	}{
		{"успешная загрузка", "/load-movies?from=1&to=2", 200},
		{"частичная ошибка", "/load-movies?from=1&to=3", 207},
		{"все фильмы отсутствуют", "/load-movies?from=3&to=4", 404},
		{"фильм отсутствует в БД", "/load-movies?from=5&to=5", 404},
		{"нет from", "/load-movies?to=2", 400},
		{"нет to", "/load-movies?from=1", 400},
		{"некорректное число", "/load-movies?from=abc&to=2", 400},
		{"нулевой ID", "/load-movies?from=0&to=2", 400},
		{"обратный диапазон", "/load-movies?from=5&to=2", 400},
		{"дублированный параметр", "/load-movies?from=1&from=2&to=3", 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("ожидали HTTP %d, получили %d: %s",
					tt.status, rec.Code, rec.Body.String())
			}

			if rec.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("неверный Content-Type")
			}
		})
	}

	t.Run("формат успешного ответа", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/load-movies?from=1&to=2", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		var movies []movie.Movie
		if err := json.Unmarshal(rec.Body.Bytes(), &movies); err != nil {
			t.Fatalf("ожидали JSON-массив: %v", err)
		}

		if len(movies) != 2 || movies[0].ID != 1 || movies[1].ID != 2 {
			t.Fatalf("неверный список фильмов: %+v", movies)
		}
	})

	t.Run("формат частичных ошибок", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/load-movies?from=1&to=3", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		var result movie.RangeResult
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("неверный JSON: %v", err)
		}

		if len(result.Movies) != 2 || len(result.Errors) != 1 {
			t.Fatalf("неверный частичный ответ: %+v", result)
		}
		if result.Errors[0].ID != 3 {
			t.Fatalf("ожидали ошибку фильма 3")
		}
	})
}

func TestGetLastMovieHandler(t *testing.T) {
	mux := newTestMux(t)

	req := httptest.NewRequest(http.MethodGet, "/last-movie", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали HTTP 200, получили %d", rec.Code)
	}

	var result movie.Movie
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("неверный JSON: %v", err)
	}

	if result.ID != 2 {
		t.Fatalf("ожидали последний фильм ID=2, получили %d", result.ID)
	}
}

func TestHTTPMethod(t *testing.T) {
	mux := newTestMux(t)

	req := httptest.NewRequest(http.MethodPost, "/last-movie", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("ожидали HTTP 405, получили %d", rec.Code)
	}
}
