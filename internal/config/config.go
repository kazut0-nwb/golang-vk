package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	DefaultMovieBaseURL = "https://homeworksite.site"
	DefaultHTTPAddr     = ":8080"
	DefaultSyncWorkers  = 5
)

type Config struct {
	MovieBaseURL string
	HTTPAddr     string
	DatabaseURL  string
	RedisAddr    string
	SyncWorkers  int
}

func Load() Config {
	return Config{
		MovieBaseURL: getEnv("MOVIE_BASE_URL", DefaultMovieBaseURL),
		HTTPAddr:     getEnv("HTTP_ADDR", DefaultHTTPAddr),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		RedisAddr:    os.Getenv("REDIS_ADDR"),
		SyncWorkers:  getEnvInt("SYNC_WORKERS", DefaultSyncWorkers),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}

	result, err := strconv.Atoi(value)
	if err != nil || result <= 0 {
		return defaultValue
	}

	return result
}
