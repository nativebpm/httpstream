package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// LoggingMiddleware returns a middleware that logs HTTP requests and responses using slog.
func LoggingMiddleware(logger *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return &loggingRoundTripper{
			next:   next,
			logger: logger,
		}
	}
}

type loggingRoundTripper struct {
	next   http.RoundTripper
	logger *slog.Logger
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	// Log the request
	l.logger.Info("HTTP Request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", req.Header,
	)

	// Perform the request
	resp, err := l.next.RoundTrip(req)
	duration := time.Since(start)

	if err != nil {
		l.logger.Error("HTTP Request failed",
			"method", req.Method,
			"url", req.URL.String(),
			"duration", duration,
			"error", err,
		)
		return resp, err
	}

	// Log the response
	l.logger.Info("HTTP Response",
		"method", req.Method,
		"url", req.URL.String(),
		"status", resp.StatusCode,
		"duration", duration,
		"headers", resp.Header,
	)

	return resp, nil
}
