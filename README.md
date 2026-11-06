# Movie API

Go-сервис для работы с фильмами через HTTP и gRPC API.

## Возможности
- Хранение фильмов в PostgreSQL.
- Автоматическая синхронизация с внешним фильм-стендом при запуске.
- Поиск последнего доступного фильма.
- Защита от дублирования записей.
- HTTP и gRPC API для получения фильмов.
- Swagger-документация.

## Технологии
- Go, PostgreSQL, pgx/v5.
- gRPC, Protocol Buffers.
- Swagger/OpenAPI.
- Docker Compose, SQL migrations (Goose).

## Запуск

Запусти PostgreSQL:
```bash
docker compose up -d
```

Настрой переменные окружения: `DATABASE_URL`, `MOVIE_BASE_URL`, `HTTP_ADDR`.

Запусти приложение:
```bash
go run ./cmd/server
```

При старте автоматически применяются миграции и выполняется синхронизация фильмов.

## API
- HTTP: `localhost:8080`
- gRPC: `localhost:50051`
- Swagger: `http://localhost:8080/swagger/index.html`
