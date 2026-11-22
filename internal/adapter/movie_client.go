package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"my-go-app/internal/domain"
)

const (
	maxFetchAttempts = 3
	retryBaseDelay   = 200 * time.Millisecond
)

type UpstreamStatusError struct {
	StatusCode int
}

func (e *UpstreamStatusError) Error() string {
	return fmt.Sprintf(
		"внешний стенд вернул HTTP %d",
		e.StatusCode,
	)
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

// FetchMovie получает фильм с внешнего стенда.
// При временных ошибках выполняет повторные попытки.
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

	url := fmt.Sprintf(
		"%s/%d/info.0.json",
		c.baseURL,
		id,
	)

	var lastErr error

	for attempt := 1; attempt <= maxFetchAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		movie, retry, err := c.fetchAttempt(ctx, url, id)

		if err == nil {
			return movie, nil
		}

		lastErr = err

		if !retry || attempt == maxFetchAttempts {
			return nil, err
		}

		if err := waitForRetry(
			ctx,
			retryDelay(attempt),
		); err != nil {
			return nil, err
		}
	}

	return nil, lastErr
}

// fetchAttempt выполняет одну HTTP-попытку.
// Возвращает retry=true, если запрос имеет смысл повторить.
func (c *Client) fetchAttempt(
	ctx context.Context,
	url string,
	id int,
) (*domain.Movie, bool, error) {
	reqCtx, cancel := context.WithTimeout(
		ctx,
		c.timeout,
	)
	defer cancel()

	req, err := http.NewRequestWithContext(
		reqCtx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, false, fmt.Errorf(
			"не удалось создать HTTP-запрос: %w",
			err,
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}

		return nil,
			isRetryableNetworkError(err),
			fmt.Errorf(
				"не удалось получить фильм %d: %w",
				id,
				err,
			)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, fmt.Errorf(
			"фильм %d: %w",
			id,
			domain.ErrMovieNotFound,
		)
	}

	if resp.StatusCode != http.StatusOK {
		statusErr := &UpstreamStatusError{
			StatusCode: resp.StatusCode,
		}

		return nil,
			isRetryableStatus(resp.StatusCode),
			statusErr
	}

	var result domain.Movie

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		retry := isRetryableBodyError(err)

		return nil, retry, fmt.Errorf(
			"не удалось прочитать JSON фильма %d: %w",
			id,
			err,
		)
	}

	if result.ID == 0 {
		result.ID = id
	}

	return &result, false, nil
}

// isRetryableStatus определяет временные HTTP-ошибки.
func isRetryableStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status >= 500 && status <= 599
}

// isRetryableNetworkError определяет временные сетевые ошибки.
func isRetryableNetworkError(err error) bool {
	if errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error

	return errors.As(err, &netErr)
}

// isRetryableBodyError проверяет ошибки чтения HTTP-ответа.
func isRetryableBodyError(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		isRetryableNetworkError(err)
}

// retryDelay возвращает задержку между попытками.
func retryDelay(attempt int) time.Duration {
	return retryBaseDelay * time.Duration(1<<(attempt-1))
}

// waitForRetry ожидает задержку с поддержкой отмены контекста.
func waitForRetry(
	ctx context.Context,
	delay time.Duration,
) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-timer.C:
		return nil
	}
}
