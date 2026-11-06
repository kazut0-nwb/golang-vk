package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("MOVIE_BASE_URL", "https://example.com")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")

	cfg := Load()

	if cfg.MovieBaseURL != "https://example.com" {
		t.Errorf("unexpected MovieBaseURL: %s", cfg.MovieBaseURL)
	}

	if cfg.HTTPAddr != ":9090" {
		t.Errorf("unexpected HTTPAddr: %s", cfg.HTTPAddr)
	}

	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL should not be empty")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MOVIE_BASE_URL", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")

	cfg := Load()

	if cfg.MovieBaseURL != DefaultMovieBaseURL {
		t.Errorf("unexpected default MovieBaseURL: %s", cfg.MovieBaseURL)
	}

	if cfg.HTTPAddr != DefaultHTTPAddr {
		t.Errorf("unexpected default HTTPAddr: %s", cfg.HTTPAddr)
	}

	if cfg.DatabaseURL != "" {
		t.Error("DatabaseURL should be empty when not configured")
	}
}
