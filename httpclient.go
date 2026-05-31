package httpclient

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/nativebpm/httpclient/internal/httprequest"
	"github.com/nativebpm/httpclient/internal/httptransport"
)

type method string

const (
	GET     method = http.MethodGet
	POST    method = http.MethodPost
	PUT     method = http.MethodPut
	PATCH   method = http.MethodPatch
	DELETE  method = http.MethodDelete
	HEAD    method = http.MethodHead
	CONNECT method = http.MethodConnect
	OPTIONS method = http.MethodOptions
	TRACE   method = http.MethodTrace
)

type Middleware = httptransport.Middleware
type Multipart = httprequest.Multipart
type Request = httprequest.Request

type HTTPClient struct {
	client      http.Client
	baseURL     url.URL
	middlewares []Middleware
}

func NewClient(client http.Client, baseURL string) (*HTTPClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &HTTPClient{
		client:      client,
		baseURL:     *u,
		middlewares: []Middleware{},
	}, nil
}

func (c *HTTPClient) url(path string) string {
	return c.baseURL.JoinPath(path).String()
}

func (c *HTTPClient) Use(middleware Middleware) *HTTPClient {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

func (c *HTTPClient) Request(ctx context.Context, method method, path string) *httprequest.Request {
	client := c.client
	for _, mw := range c.middlewares {
		if client.Transport == nil {
			client.Transport = http.DefaultTransport
		}
		client.Transport = mw(client.Transport)
	}
	return httprequest.NewRequest(ctx, client, string(method), c.url(path))
}

func (c *HTTPClient) MultipartRequest(ctx context.Context, method method, path string) *httprequest.Multipart {
	client := c.client
	for _, mw := range c.middlewares {
		if client.Transport == nil {
			client.Transport = http.DefaultTransport
		}
		client.Transport = mw(client.Transport)
	}
	return httprequest.NewMultipart(ctx, client, string(method), c.url(path))
}

func (c *HTTPClient) GET(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, GET, path)
}

func (c *HTTPClient) POST(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, POST, path)
}

func (c *HTTPClient) PUT(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, PUT, path)
}

func (c *HTTPClient) PATCH(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, PATCH, path)
}

func (c *HTTPClient) DELETE(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, DELETE, path)
}

func (c *HTTPClient) Multipart(ctx context.Context, path string) *httprequest.Multipart {
	return c.MultipartRequest(ctx, POST, path)
}

func (c *HTTPClient) WithLogger(logger *slog.Logger) *HTTPClient {
	return c.Use(httptransport.LoggingMiddleware(logger))
}
