package movie

import (
	"context"
	"time"

	"my-go-app/internal/adapter"
	"my-go-app/internal/domain"
)

var ErrMovieNotFound = domain.ErrMovieNotFound

type UpstreamStatusError = adapter.UpstreamStatusError

type Client struct {
	inner *adapter.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		inner: adapter.NewClient(baseURL, timeout),
	}
}

func (c *Client) FetchMovie(
	ctx context.Context,
	id int,
) (*Movie, error) {
	return c.inner.FetchMovie(ctx, id)
}
