# Movie API

HTTP и gRPC API для работы с фильмами.

## Запуск

```powershell
go run ./cmd/server
```

HTTP API
- GET /load-movies?from=1&to=3
- GET /last-movie
- Swagger UI: http://localhost:8080/swagger/index.html

gRPC API
- movies.v1.MovieService/LoadMovies
- movies.v1.MovieService/GetLastMovie

gRPC-порт: 50051.

## Структура проекта

```text
golang-vk/
├── api/proto/          # Protobuf-схемы и сгенерированный gRPC-код
├── cmd/server/         # Точка входа приложения
├── docs/               # Swagger-документация
├── internal/
│   ├── config/         # Конфигурация
│   ├── grpcapi/        # gRPC-обработчики
│   ├── httpapi/        # HTTP-обработчики
│   └── movie/          # Клиент внешнего стенда и бизнес-логика
├── go.mod
├── go.sum
└── README.md
```
