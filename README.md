# Movie API

Go-сервис для работы с фильмами, предоставляющий HTTP и gRPC API. Приложение использует PostgreSQL для хранения информации о фильмах и внешний фильм-стенд в качестве источника данных для синхронизации.

## Возможности

- Синхронизация фильмов с внешнего стенда при запуске приложения.
- Поиск последнего доступного фильма с помощью экспоненциального и бинарного поиска.
- Сохранение фильмов в PostgreSQL.
- Пропуск уже сохранённых фильмов при повторной синхронизации.
- HTTP API для получения фильмов и последнего доступного фильма.
- gRPC API на основе Protocol Buffers.
- Swagger/OpenAPI документация для HTTP API.
- Обработка ошибок, таймаутов и отмены контекста.
- Graceful shutdown HTTP- и gRPC-серверов.

## Технологии

- **Go 1.26.4**
- **PostgreSQL 17**
- **pgx/v5** — работа с PostgreSQL.
- **gRPC** — RPC API.
- **Protocol Buffers** — описание контрактов gRPC.
- **Swagger/OpenAPI** — документация HTTP API.
- **Docker Compose** — запуск PostgreSQL.

## Архитектура проекта

Проект организован с разделением ответственности между слоями:

```text
golang-vk/
├── api/
│   └── proto/                  # Protobuf-контракты и сгенерированный код
├── cmd/
│   └── server/                 # Точка входа приложения
├── docs/                       # Swagger/OpenAPI документация
├── internal/
│   ├── adapter/                # Клиент внешнего фильм-стенда
│   ├── config/                 # Конфигурация приложения
│   ├── domain/                 # Доменная модель и ошибки
│   ├── handler/                # HTTP- и gRPC-обработчики
│   ├── storage/
│   │   └── postgres/           # PostgreSQL repository
│   ├── testutil/               # Тестовые реализации хранилища
│   └── usecase/                # Бизнес-логика приложения
├── migrations/                 # SQL-миграции
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

### Основные компоненты

- **Adapter** — отвечает за получение информации о фильмах с внешнего стенда.
- **Domain** — содержит доменную модель фильма и общие ошибки.
- **Storage** — реализует взаимодействие с PostgreSQL.
- **Usecase** — содержит бизнес-логику приложения, включая синхронизацию фильмов.
- **Handler** — обрабатывает HTTP- и gRPC-запросы.

HTTP и gRPC используют общий прикладной сервис, который взаимодействует с PostgreSQL через repository.

### Синхронизация фильмов

При запуске приложения выполняются следующие действия:

1. Устанавливается соединение с PostgreSQL.
2. Создаётся клиент внешнего фильм-стенда.
3. `MovieSynchronizer` определяет последний доступный фильм.
4. Получает список фильмов, уже находящихся в базе данных.
5. Загружает только отсутствующие фильмы.
6. Сохраняет полученные данные в PostgreSQL.
7. После завершения синхронизации запускаются HTTP- и gRPC-серверы.

Для поиска последнего доступного фильма используется экспоненциальный поиск с последующим бинарным поиском.

При повторной синхронизации уже сохранённые фильмы пропускаются, что позволяет избежать повторной загрузки существующих записей.

## Требования

Для запуска приложения необходимы:

- Go версии 1.26.4 или новее.
- Docker и Docker Compose.
- Доступ к внешнему фильм-стенду.
- PostgreSQL.

## Конфигурация

Приложение использует переменные окружения.

| Переменная | Значение по умолчанию | Описание |
|---|---|---|
| `MOVIE_BASE_URL` | `https://homeworksite.site` | Базовый URL внешнего фильм-стенда |
| `HTTP_ADDR` | `:8080` | Адрес HTTP-сервера |
| `DATABASE_URL` | Обязательная | Строка подключения к PostgreSQL |

Для получения фильма с идентификатором `id` используется адрес:

```text
{MOVIE_BASE_URL}/{id}/info.0.json
```

Пример строки подключения к PostgreSQL:

```text
postgres://movies_user:movies_password@localhost:5432/movies_db?sslmode=disable
```

## Запуск приложения

### 1. Запуск PostgreSQL

В корне проекта выполни:

```bash
docker compose up -d
```

Docker Compose создаст PostgreSQL 17 и настроит:

- Базу данных: `movies_db`
- Пользователя: `movies_user`
- Пароль: `movies_password`
- Порт: `5432`

Данные PostgreSQL сохраняются в Docker volume `postgres_data`.

### 2. Настройка переменных окружения

Пример для PowerShell:

```powershell
$env:DATABASE_URL = "postgres://movies_user:movies_password@localhost:5432/movies_db?sslmode=disable"
$env:MOVIE_BASE_URL = "https://homeworksite.site"
$env:HTTP_ADDR = ":8080"
```

Пример для Linux/macOS:

```bash
export DATABASE_URL="postgres://movies_user:movies_password@localhost:5432/movies_db?sslmode=disable"
export MOVIE_BASE_URL="https://homeworksite.site"
export HTTP_ADDR=":8080"
```

### 3. Применение миграций

Для управления миграциями используется Goose.

Установи Goose:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
```

Примени миграции:

PowerShell:

```powershell
goose -dir migrations postgres "$env:DATABASE_URL" up
```

Linux/macOS:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
```

Миграция `00001_create_movies.sql` создаёт таблицу `movies` со следующими полями:

| Поле | Тип | Описание |
|---|---|---|
| `id` | INTEGER | Идентификатор фильма, первичный ключ |
| `title` | TEXT | Название фильма |
| `year` | INTEGER | Год выпуска |
| `director` | TEXT | Режиссёр |

Миграции необходимо применить до запуска приложения. Автоматическое применение миграций при старте сервиса не предусмотрено.

### 4. Запуск сервиса

Из корня проекта выполни:

```bash
go run ./cmd/server
```

При успешном запуске приложение:

1. Подключится к PostgreSQL.
2. Выполнит синхронизацию фильмов с внешним стендом.
3. Запустит HTTP-сервер.
4. Запустит gRPC-сервер.

По умолчанию:

- HTTP: `localhost:8080`
- gRPC: `localhost:50051`

Для остановки приложения используй `Ctrl+C`.

## HTTP API

### GET /load-movies

Получение фильмов из PostgreSQL по диапазону идентификаторов.

Параметры:

| Параметр | Тип | Описание |
|---|---|---|
| `from` | integer | Начальный ID |
| `to` | integer | Конечный ID |

Пример запроса:

```http
GET http://localhost:8080/load-movies?from=1&to=10
```

Пример ответа:

```json
[
  {
    "id": 1,
    "title": "Movie 1",
    "year": 2020,
    "director": "Director 1"
  }
]
```

Если часть фильмов отсутствует в базе данных, возвращается HTTP `207 Multi-Status` с двумя массивами: `movies` и `errors`.

### GET /last-movie

Получение фильма с максимальным ID из PostgreSQL.

Пример запроса:

```http
GET http://localhost:8080/last-movie
```

Пример ответа:

```json
{
  "id": 10,
  "title": "Movie 10",
  "year": 2024,
  "director": "Director 10"
}
```

### HTTP-коды ответов

| Код | Описание |
|---|---|
| `200 OK` | Успешное выполнение запроса |
| `207 Multi-Status` | Частичный результат загрузки |
| `400 Bad Request` | Некорректные параметры запроса |
| `404 Not Found` | Фильм не найден |
| `502 Bad Gateway` | Ошибка при обращении к зависимостям |
| `504 Gateway Timeout` | Превышено время ожидания |

## gRPC API

gRPC-сервис описан в `api/proto/movies.proto`.

Сервис `movies.v1.MovieService` предоставляет два метода.

### LoadMovies

```protobuf
rpc LoadMovies(LoadMoviesRequest) returns (LoadMoviesResponse);
```

Загружает фильмы из PostgreSQL в указанном диапазоне ID.

Запрос:

```protobuf
message LoadMoviesRequest {
  int32 from = 1;
  int32 to = 2;
}
```

Ответ содержит список фильмов и список ошибок.

### GetLastMovie

```protobuf
rpc GetLastMovie(GetLastMovieRequest) returns (Movie);
```

Возвращает фильм с максимальным ID из PostgreSQL.

### Обработка ошибок gRPC

Для обработки ошибок используются стандартные коды:

- `InvalidArgument` — некорректные аргументы.
- `NotFound` — фильм не найден.
- `Canceled` — запрос отменён.
- `DeadlineExceeded` — превышено время ожидания.
- `Unavailable` — ошибка внешней зависимости или хранилища.

## Swagger

Документация HTTP API доступна через Swagger UI:

[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

Swagger позволяет просматривать доступные HTTP-методы, параметры запросов и структуру ответов.

## Тестирование

Для запуска всех тестов выполни:

```bash
go test ./...
```

Для запуска тестов с подробным выводом:

```bash
go test ./... -v
```

Для проверки использования конкурентного доступа:

```bash
go test -race ./...
```

Для статического анализа:

```bash
go vet ./...
```

Интеграционные тесты PostgreSQL используют переменную `DATABASE_URL`. Если она не задана, тесты repository пропускаются.

**Важно:** интеграционные тесты PostgreSQL очищают таблицу `movies` перед выполнением. Для их запуска используй отдельную тестовую базу данных, а не базу с рабочими данными.

## Остановка PostgreSQL

Для остановки контейнеров:

```bash
docker compose down
```

Для удаления контейнеров вместе с сохранёнными данными:

```bash
docker compose down -v
```

**Внимание:** команда `docker compose down -v` удаляет Docker volume с данными PostgreSQL.
