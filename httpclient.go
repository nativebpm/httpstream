package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nativebpm/httpclient/httprequest"
)

type method string

const (
	GET     method = http.MethodGet
	POST    method = http.MethodPost
	PUT     method = http.MethodPut
	PATCH   method = http.MethodPatch
	DELETE  method = http.MethodDelete
	HEAD    method = http.MethodHead
	OPTIONS method = http.MethodOptions
)

type Middleware func(http.RoundTripper) http.RoundTripper

type HTTPClient struct {
	http.Client
	baseURL     url.URL
	middlewares []Middleware
}

func NewClient(client http.Client, baseURL string) (*HTTPClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &HTTPClient{
		Client:      client,
		baseURL:     *u,
		middlewares: []Middleware{},
	}, nil
}

func (c *HTTPClient) Use(middleware Middleware) *HTTPClient {
	c.middlewares = append(c.middlewares, middleware)
	return c
}

func (c *HTTPClient) url(path string) string {
	return c.baseURL.JoinPath(path).String()
}

func (c *HTTPClient) NewRequest(ctx context.Context, method method, path string) *httprequest.Request {
	client := c.Client
	for _, mw := range c.middlewares {
		if client.Transport == nil {
			client.Transport = http.DefaultTransport
		}
		client.Transport = mw(client.Transport)
	}
	return httprequest.NewRequest(ctx, client, string(method), c.url(path))
}

func (c *HTTPClient) NewMultipart(ctx context.Context, method method, path string) *httprequest.Multipart {
	client := c.Client
	for _, mw := range c.middlewares {
		if client.Transport == nil {
			client.Transport = http.DefaultTransport
		}
		client.Transport = mw(client.Transport)
	}
	return httprequest.NewMultipart(ctx, client, string(method), c.url(path))
}
