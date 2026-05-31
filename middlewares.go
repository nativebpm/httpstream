package httpstream

import (
	"log/slog"
	"net/http"

	"github.com/nativebpm/httpstream/internal/httptransport"
)

func LoggingMiddleware(logger *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	return httptransport.LoggingMiddleware(logger)
}

// ConcurrencyMiddleware is a convenience wrapper that exposes the internal
// concurrency limiter middleware for external packages. It returns a
// Middleware that limits the number of concurrent in-flight HTTP requests.
func ConcurrencyMiddleware(limit int) func(http.RoundTripper) http.RoundTripper {
	return httptransport.ConcurrencyMiddleware(limit)
}
