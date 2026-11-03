package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"my-go-app/internal/domain"
)

type UpstreamStatusError struct {
	StatusCode int
}

func (e *UpstreamStatusError) Error() string {
	return fmt.Sprintf("внешний стенд вернул HTTP %d", e.StatusCode)
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	timeout    time.Duration
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
		timeout:    timeout,
	}
}

func (c *Client) FetchMovie(
	ctx context.Context,
	id int,
) (*domain.Movie, error) {
	if id < 1 {
		return nil, fmt.Errorf(
			"id фильма должен быть >= 1, получено: %d",
			id,
		)
	}

	reqCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	url := fmt.Sprintf("%s/%d/info.0.json", c.baseURL, id)

	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"не удалось создать HTTP-запрос: %w",
			err,
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"не удалось получить фильм %d: %w",
			id,
			err,
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf(
			"фильм %d: %w",
			id,
			domain.ErrMovieNotFound,
		)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &UpstreamStatusError{
			StatusCode: resp.StatusCode,
		}
	}

	var result domain.Movie

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf(
			"не удалось прочитать JSON фильма %d: %w",
			id,
			err,
		)
	}

	if result.ID == 0 {
		result.ID = id
	}

	return &result, nil
}
