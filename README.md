# Movie API

Go-сервис для хранения, синхронизации и поиска фильмов через HTTP и gRPC API.

## Возможности

- PostgreSQL — основное хранилище фильмов.
- Автоматическая синхронизация при запуске с защитой от дубликатов.
- Параллельная загрузка фильмов с настраиваемым количеством workers.
- Обратный индекс Redis по названию и режиссёру.
- Поиск по одному или нескольким словам без учёта регистра.
- Кэширование поисковых ответов в Redis на 3600 секунд.

## Запуск

Запусти PostgreSQL и Redis:

```bash
docker compose up -d
```

Настрой переменные окружения:

```powershell
$env:DATABASE_URL = "postgres://movies_user:movies_password@localhost:5432/movies_db?sslmode=disable"
$env:REDIS_ADDR = "localhost:6379"
$env:MOVIE_BASE_URL = "https://homeworksite.site"
$env:HTTP_ADDR = ":8080"
$env:SYNC_WORKERS = "5"
```

Запусти приложение из корня Go-модуля:

```bash
go run ./cmd/server
```

При запуске автоматически применяются миграции Goose, выполняется синхронизация фильмов и построение индекса Redis по всем фильмам из PostgreSQL.

## Поиск и кэширование

Обратный индекс хранится в Redis Set по ключам `movies:index:word:<word>`, а для многословного поиска используется логика AND через `SINTER`.

Полные записи найденных фильмов загружаются из PostgreSQL, после чего успешный JSON-ответ кэшируется в Redis на 3600 секунд без продления TTL при чтении.

При перестроении индекса старый поисковый кэш сбрасывается, а после очистки Redis индекс автоматически восстанавливается при следующем запуске.

## API

- HTTP: `localhost:8080`
- gRPC: `localhost:50051`
- Swagger: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
- Поиск: `GET /search?q=Solaris`
- Другие методы: `GET /load-movies`, `GET /last-movie`.

## Проверка

```bash
go test ./...
go test -race ./...
```