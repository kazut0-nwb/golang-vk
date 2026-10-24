package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	moviepb "my-go-app/api/proto"
	"my-go-app/internal/config"
	"my-go-app/internal/grpcapi"
	"my-go-app/internal/httpapi"
	"my-go-app/internal/movie"

	"google.golang.org/grpc"

	httpSwagger "github.com/swaggo/http-swagger/v2"

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

	// HTTP API и Swagger.
	mux := http.NewServeMux()

	handler := httpapi.NewHandler(service)
	handler.RegisterRoutes(mux)

	mux.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// gRPC API.
	const grpcAddr = ":50051"

	listener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("не удалось открыть gRPC-порт %s: %v", grpcAddr, err)
	}

	grpcServer := grpc.NewServer()
	moviepb.RegisterMovieServiceServer(
		grpcServer,
		grpcapi.NewServer(service),
	)

	// Корректное завершение обоих серверов.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	errCh := make(chan error, 2)

	go func() {
		log.Printf("HTTP-сервер запускается на %s", cfg.HTTPAddr)

		if err := httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("HTTP-сервер: %w", err)
		}
	}()

	go func() {
		log.Printf("gRPC-сервер запускается на %s", grpcAddr)

		if err := grpcServer.Serve(listener); err != nil &&
			!errors.Is(err, grpc.ErrServerStopped) {
			errCh <- fmt.Errorf("gRPC-сервер: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("получен сигнал остановки")

	case err := <-errCh:
		log.Printf("ошибка работы сервера: %v", err)
		stop()
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("ошибка остановки HTTP-сервера: %v", err)
	}

	cancel()

	grpcDone := make(chan struct{})

	go func() {
		grpcServer.GracefulStop()
		close(grpcDone)
	}()

	grpcCtx, grpcCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer grpcCancel()

	select {
	case <-grpcDone:
		log.Println("gRPC-сервер остановлен")

	case <-grpcCtx.Done():
		grpcServer.Stop()
		log.Println("gRPC-сервер принудительно остановлен")
	}
}
