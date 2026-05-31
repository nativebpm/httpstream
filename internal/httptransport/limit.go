package httptransport

import (
	"net/http"
)

// ConcurrencyMiddleware returns a Middleware that limits the number of
// concurrent HTTP requests in flight. It uses a buffered channel as a semaphore.
// When limit <= 0, the middleware is a no-op.
func ConcurrencyMiddleware(limit int) func(http.RoundTripper) http.RoundTripper {
	if limit <= 0 {
		return func(next http.RoundTripper) http.RoundTripper { return next }
	}

	sem := make(chan struct{}, limit)

	return func(next http.RoundTripper) http.RoundTripper {
		return &concurrencyLimiter{next: next, sem: sem}
	}
}

type concurrencyLimiter struct {
	next http.RoundTripper
	sem  chan struct{}
}

func (c *concurrencyLimiter) RoundTrip(req *http.Request) (*http.Response, error) {
	// Acquire slot
	c.sem <- struct{}{}
	defer func() { <-c.sem }()
	return c.next.RoundTrip(req)
}
