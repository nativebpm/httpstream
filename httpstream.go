package httpstream

import (
	"context"
	"net/http"
	"net/url"

	"github.com/nativebpm/httpstream/internal/httprequest"
)

type HttpMethod string

const (
	GET     HttpMethod = http.MethodGet
	POST    HttpMethod = http.MethodPost
	PUT     HttpMethod = http.MethodPut
	PATCH   HttpMethod = http.MethodPatch
	DELETE  HttpMethod = http.MethodDelete
	HEAD    HttpMethod = http.MethodHead
	CONNECT HttpMethod = http.MethodConnect
	OPTIONS HttpMethod = http.MethodOptions
	TRACE   HttpMethod = http.MethodTrace
)

type Multipart = httprequest.Multipart
type Request = httprequest.Request

type Client struct {
	HttpClient http.Client
	BaseURL    url.URL
}

func NewClient(client *http.Client, baseURL string) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &Client{HttpClient: *client, BaseURL: *u}, nil
}

func (c *Client) url(path string) string {
	return c.BaseURL.JoinPath(path).String()
}

func (c *Client) Use(middleware func(http.RoundTripper) http.RoundTripper) *Client {
	if c.HttpClient.Transport == nil {
		c.HttpClient.Transport = http.DefaultTransport
	}
	c.HttpClient.Transport = middleware(c.HttpClient.Transport)
	return c
}

func (c *Client) Request(ctx context.Context, method HttpMethod, path string) *httprequest.Request {
	return httprequest.NewRequest(ctx, c.HttpClient, string(method), c.url(path))
}

func (c *Client) MultipartRequest(ctx context.Context, method HttpMethod, path string) *httprequest.Multipart {
	return httprequest.NewMultipart(ctx, c.HttpClient, string(method), c.url(path))
}

func (c *Client) GET(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, GET, path)
}

func (c *Client) POST(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, POST, path)
}

func (c *Client) PUT(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, PUT, path)
}

func (c *Client) PATCH(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, PATCH, path)
}

func (c *Client) DELETE(ctx context.Context, path string) *httprequest.Request {
	return c.Request(ctx, DELETE, path)
}

func (c *Client) Multipart(ctx context.Context, path string) *httprequest.Multipart {
	return c.MultipartRequest(ctx, POST, path)
}
