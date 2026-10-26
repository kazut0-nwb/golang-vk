# Movie API

HTTP и gRPC API для загрузки информации о фильмах с внешнего стенда.

## Возможности

- Загрузка фильмов по диапазону ID.
- Поиск последнего доступного фильма.
- HTTP API с JSON-ответами.
- gRPC API на основе Protocol Buffers.
- Документация Swagger/OpenAPI.
- Обработка ошибок внешнего стенда, таймаутов и частичных результатов.

## Требования

- Go
- Доступ к внешнему фильм-стенду (если используются реальные данные).

## Запуск

Из корня проекта выполни:

```powershell
go run ./cmd/server
```

По умолчанию запускаются:

- HTTP-сервер на `:8080`.
- gRPC-сервер на `:50051`.

Для остановки приложения нажми `Ctrl+C`.

## Конфигурация

Настройки передаются через переменные окружения.

| Переменная | Значение по умолчанию | Назначение |
|---|---|---|
| `MOVIE_BASE_URL` | `https://homeworksite.site` | Адрес внешнего фильм-стенда |
| `HTTP_ADDR` | `:8080` | Адрес HTTP-сервера |

Для каждого фильма с ID `id` клиент обращается к адресу:

```text
{MOVIE_BASE_URL}/{id}/info.0.json
```

Пример настройки в PowerShell:

```powershell
$env:MOVIE_BASE_URL = "https://homeworksite.site"
$env:HTTP_ADDR = ":8080"
go run ./cmd/server
```

## HTTP API

### `GET /load-movies?from=1&to=3`

Загружает фильмы в диапазоне ID от `from` до `to` включительно.

Пример:

```powershell
Invoke-RestMethod "http://localhost:8080/load-movies?from=1&to=3"
```

При успешной загрузке всех фильмов возвращается JSON-массив фильмов.

При частичных ошибках возвращается HTTP `207 Multi-Status` с массивами `movies` и `errors`.

Основные коды ответов:

- `200 OK` — все фильмы загружены успешно.
- `207 Multi-Status` — получены частичные результаты.
- `400 Bad Request` — некорректные параметры диапазона.
- `404 Not Found` — фильмы не найдены.
- `502 Bad Gateway` — ошибка внешнего стенда.
- `504 Gateway Timeout` — превышено время ожидания.

### `GET /last-movie`

Находит последний доступный фильм и возвращает его в JSON.

Пример:

```powershell
Invoke-RestMethod "http://localhost:8080/last-movie"
```

### Swagger UI

Документация HTTP API доступна по адресу:

http://localhost:8080/swagger/index.html

## gRPC API

gRPC-сервис `movies.v1.MovieService` предоставляет методы:

- `LoadMovies(LoadMoviesRequest) returns (LoadMoviesResponse)` — загрузка диапазона фильмов.
- `GetLastMovie(GetLastMovieRequest) returns (Movie)` — поиск последнего фильма.

Protobuf-контракт находится в `api/proto/movies.proto`.

gRPC-сервер по умолчанию слушает порт `50051`.

## Архитектура проекта

```text
golang-vk/
├── api/
│   └── proto/                 # Protobuf-контракт и сгенерированный код
├── cmd/
│   └── server/                # Точка входа приложения
├── docs/                      # Swagger/OpenAPI
├── internal/
│   ├── adapter/                # Клиент внешнего фильм-стенда
│   ├── config/                 # Конфигурация приложения
│   ├── domain/                 # Доменная модель и ошибки
│   ├── handler/                # HTTP- и gRPC-обработчики
│   ├── usecase/                # Прикладная бизнес-логика
│   ├── grpcapi/                # Совместимый код и тесты предыдущей реализации
│   ├── httpapi/                # Совместимый код и регрессионные тесты
│   └── movie/                  # Совместимые типы, обёртки и тесты
├── go.mod
├── go.sum
└── README.md
```

Основные зависимости приложения собираются в `cmd/server/main.go`.

Бизнес-логика находится в `internal/usecase`, доменная модель — в `internal/domain`, взаимодействие с внешним стендом — в `internal/adapter`. HTTP- и gRPC-обработчики используют общий прикладной сервис.

## Проверки

Запустить все тесты:

```powershell
go test ./...
```

Выполнить статическую проверку:

```powershell
go vet ./...
```
