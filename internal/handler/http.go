package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"my-go-app/internal/domain"
	"my-go-app/internal/usecase"
)

type HTTPHandler struct {
	service *usecase.Service
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewHTTPHandler(service *usecase.Service) *HTTPHandler {
	return &HTTPHandler{service: service}
}

func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /load-movies", h.LoadMovies)
	mux.HandleFunc("GET /last-movie", h.GetLastMovie)
}

func parseQueryInt(r *http.Request, name string) (int, error) {
	values := r.URL.Query()[name]

	if len(values) != 1 || values[0] == "" {
		return 0, fmt.Errorf(
			"параметр %q должен быть указан один раз", name,
		)
	}

	value, err := strconv.Atoi(values[0])
	if err != nil {
		return 0, fmt.Errorf(
			"параметр %q должен быть целым числом", name,
		)
	}

	return value, nil
}

// LoadMovies godoc
// @Summary Загрузить фильмы по диапазону ID
// @Description Загружает фильмы с внешнего стенда от from до to включительно.
// @Description При частичных ошибках возвращает HTTP 207 и списки movies и errors.
// @Tags Movies
// @Produce json
// @Param from query int true "Начальный ID фильма, от 1"
// @Param to query int true "Конечный ID фильма, не меньше from"
// @Success 200 {array} domain.Movie
// @Success 207 {object} usecase.RangeResult
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} usecase.RangeResult
// @Failure 502 {object} usecase.RangeResult
// @Failure 504 {object} ErrorResponse
// @Router /load-movies [get]
func (h *HTTPHandler) LoadMovies(
	w http.ResponseWriter,
	r *http.Request,
) {
	from, err := parseQueryInt(r, "from")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	to, err := parseQueryInt(r, "to")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.LoadMovies(r.Context(), from, to)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	if len(result.Errors) == 0 {
		writeJSON(w, http.StatusOK, result.Movies)
		return
	}

	if len(result.Movies) > 0 {
		writeJSON(w, http.StatusMultiStatus, result)
		return
	}

	allNotFound := true
	for _, item := range result.Errors {
		if !errors.Is(item.Cause, domain.ErrMovieNotFound) {
			allNotFound = false
			break
		}
	}

	if allNotFound {
		writeJSON(w, http.StatusNotFound, result)
		return
	}

	writeJSON(w, http.StatusBadGateway, result)
}

// GetLastMovie godoc
// @Summary Получить последний доступный фильм
// @Description Находит фильм с максимальным ID на внешнем стенде.
// @Tags Movies
// @Produce json
// @Success 200 {object} domain.Movie
// @Failure 404 {object} ErrorResponse
// @Failure 504 {object} ErrorResponse
// @Router /last-movie [get]
func (h *HTTPHandler) GetLastMovie(
	w http.ResponseWriter,
	r *http.Request,
) {
	result, err := h.service.GetLastMovie(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidRange):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, domain.ErrMovieNotFound):
		writeError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, context.DeadlineExceeded):
		writeError(
			w,
			http.StatusGatewayTimeout,
			"превышено время ожидания",
		)

	case errors.Is(err, context.Canceled):
		writeError(
			w,
			http.StatusRequestTimeout,
			"запрос отменён",
		)

	default:
		writeError(
			w,
			http.StatusBadGateway,
			fmt.Sprintf("ошибка фильм-стенда: %v", err),
		)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	body, err := json.Marshal(data)
	if err != nil {
		http.Error(
			w,
			"не удалось сформировать JSON",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	_, _ = w.Write(append(body, '\n'))
}
