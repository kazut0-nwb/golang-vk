package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("MOVIE_BASE_URL", "https://example.com")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("SYNC_WORKERS", "10")

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

	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("unexpected RedisAddr: %s", cfg.RedisAddr)
	}

	if cfg.RedisAddr != "localhost:6379" {
		t.Errorf("unexpected RedisAddr: %s", cfg.RedisAddr)
	}

	if cfg.SyncWorkers != 10 {
		t.Errorf("unexpected SyncWorkers: %d", cfg.SyncWorkers)
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MOVIE_BASE_URL", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("REDIS_ADDR", "")
	t.Setenv("SYNC_WORKERS", "")

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

	if cfg.RedisAddr != "" {
		t.Error("RedisAddr should be empty when not configured")
	}

	if cfg.RedisAddr != "" {
		t.Error("RedisAddr should be empty when not configured")
	}

	if cfg.SyncWorkers != DefaultSyncWorkers {
		t.Errorf(
			"unexpected default SyncWorkers: %d",
			cfg.SyncWorkers,
		)
	}
}

func TestLoadInvalidSyncWorkers(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "zero",
			value: "0",
		},
		{
			name:  "negative",
			value: "-5",
		},
		{
			name:  "not a number",
			value: "abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SYNC_WORKERS", tt.value)

			cfg := Load()

			if cfg.SyncWorkers != DefaultSyncWorkers {
				t.Errorf(
					"expected default %d, got %d",
					DefaultSyncWorkers,
					cfg.SyncWorkers,
				)
			}
		})
	}
}
