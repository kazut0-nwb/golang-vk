# Movie API

HTTP и gRPC API для работы с фильмами.

## Запуск

```powershell
go run ./cmd/server

HTTP API
- GET /load-movies?from=1&to=3
- GET /last-movie
- Swagger UI: http://localhost:8080/swagger/index.html

gRPC API
- movies.v1.MovieService/LoadMovies
- movies.v1.MovieService/GetLastMovie

gRPC-порт: 50051.