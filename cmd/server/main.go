package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	"my-go-app/internal/config"
	"my-go-app/internal/httpapi"
	"my-go-app/internal/movie"

	_ "my-go-app/docs"
)

// @title Movie API
// @version 1.0
// @description HTTP API для загрузки фильмов с внешнего стенда.
// @BasePath /
func main() {
	cfg := config.Load()

	client := movie.NewClient(
		cfg.MovieBaseURL,
		5*time.Second,
	)
	service := movie.NewService(client)

	mux := http.NewServeMux()

	handler := httpapi.NewHandler(service)
	handler.RegisterRoutes(mux)

	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf(
				"ошибка остановки HTTP-сервера: %v",
				err,
			)
		}
	}()

	log.Printf("HTTP-сервер запускается на %s", cfg.HTTPAddr)

	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("ошибка HTTP-сервера: %v", err)
	}
}
