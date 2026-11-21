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
	"my-go-app/internal/adapter"
	"my-go-app/internal/config"
	"my-go-app/internal/handler"
	"my-go-app/internal/storage/postgres"
	"my-go-app/internal/usecase"

	"google.golang.org/grpc"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "my-go-app/docs"

	redisstore "my-go-app/internal/storage/redis"
)

// @title Movie API
// @version 1.0
// @description HTTP API для загрузки фильмов с внешнего стенда.
// @BasePath /
func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Подключение к PostgreSQL.
	dbCtx, dbCancel := context.WithTimeout(
		ctx,
		30*time.Second,
	)

	pool, err := postgres.NewPool(dbCtx, cfg.DatabaseURL)
	dbCancel()

	if err != nil {
		log.Fatalf("ошибка подключения к PostgreSQL: %v", err)
	}

	defer pool.Close()

	log.Println("успешное подключение к PostgreSQL")

	// Применение миграций перед использованием базы данных.
	migrationCtx, migrationCancel := context.WithTimeout(
		ctx,
		30*time.Second,
	)

	if err := postgres.RunMigrations(migrationCtx, pool); err != nil {
		migrationCancel()
		log.Fatalf("ошибка применения миграций PostgreSQL: %v", err)
	}

	migrationCancel()

	log.Println("миграции PostgreSQL успешно применены")

	movieRepository := postgres.NewMovieRepository(pool)

	client := adapter.NewClient(
		cfg.MovieBaseURL,
		15*time.Second,
	)

	// Подключение к Redis.
	redisCtx, redisCancel := context.WithTimeout(
		ctx,
		5*time.Second,
	)

	redisClient, err := redisstore.NewClient(
		redisCtx,
		cfg.RedisAddr,
	)

	redisCancel()

	if err != nil {
		log.Fatalf("ошибка подключения к Redis: %v", err)
	}

	defer redisClient.Close()

	log.Println("успешное подключение к Redis")

	// Синхронизация фильмов с внешнего стенда.
	synchronizer := usecase.NewMovieSynchronizerWithWorkers(
		client,
		movieRepository,
		cfg.SyncWorkers,
	)

	log.Printf(
		"количество workers синхронизации: %d",
		cfg.SyncWorkers,
	)

	log.Println("начинается синхронизация фильмов...")

	if err := synchronizer.Sync(ctx); err != nil {
		pool.Close()
		log.Fatalf("ошибка синхронизации фильмов: %v", err)
	}

	log.Println("синхронизация фильмов завершена")

	indexCtx, indexCancel := context.WithTimeout(
		ctx,
		30*time.Second,
	)

	movies, err := movieRepository.GetMovies(indexCtx)
	if err != nil {
		indexCancel()
		log.Fatalf("не удалось получить фильмы для индексации: %v", err)
	}

	movieIndex := redisstore.NewIndex(redisClient)

	if err := movieIndex.Rebuild(indexCtx, movies); err != nil {
		indexCancel()
		log.Fatalf("ошибка построения индекса Redis: %v", err)
	}

	indexCancel()

	log.Printf(
		"обратный индекс Redis построен: %d фильмов",
		len(movies),
	)

	// Создаём Redis-кэш поисковых ответов.
	searchCache := redisstore.NewCache(redisClient)

	// Создаём поисковый сервис.
	searchService := usecase.NewSearchService(
		movieRepository,
		movieIndex,
		searchCache,
	)

	// Существующий сервис ДЗ5 оставляем без изменений.
	service := usecase.NewService(movieRepository)

	// HTTP API и Swagger.
	mux := http.NewServeMux()

	httpHandler := handler.NewHTTPHandler(
		service,
		searchService,
	)
	httpHandler.RegisterRoutes(mux)

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
		handler.NewGRPCServer(service),
	)

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
