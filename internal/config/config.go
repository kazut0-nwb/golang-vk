package config

import "os"

const (
	DefaultMovieBaseURL = "https://homeworksite.site"
	DefaultHTTPAddr     = ":8080"
)

type Config struct {
	MovieBaseURL string
	HTTPAddr     string
}

func Load() Config {
	return Config{
		MovieBaseURL: getEnv("MOVIE_BASE_URL", DefaultMovieBaseURL),
		HTTPAddr:     getEnv("HTTP_ADDR", DefaultHTTPAddr),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}
