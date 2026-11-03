package domain

import "errors"

var (
	ErrInvalidRange  = errors.New("некорректный диапазон фильмов")
	ErrMovieNotFound = errors.New("фильм не найден на стенде")
)

type Movie struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Director string `json:"director"`
}
