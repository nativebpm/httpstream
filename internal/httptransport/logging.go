package httptransport

import (
	"log/slog"
	"net/http"
	"time"
)

// LoggingMiddleware returns a Middleware that logs HTTP requests and responses
// using the provided *slog.Logger. It is compatible with the Middleware type
// expected by the package: func(http.RoundTripper) http.RoundTripper.
//
// The middleware logs an entry before the request is sent and after the
// response is received (or when an error occurs). It records method, url,
// duration, status (when available), headers and the error when present.
func LoggingMiddleware(logger *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	if logger == nil {
		// Use the default logger if nil was provided to avoid panics.
		logger = slog.Default()
	}

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

	// Log request start
	l.logger.Info("HTTP Request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", req.Header,
	)

	// Delegate to the next RoundTripper
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

	// Log response details
	l.logger.Info("HTTP Response",
		"method", req.Method,
		"url", req.URL.String(),
		"status", resp.StatusCode,
		"duration", duration,
		"headers", resp.Header,
	)

	return resp, nil
}
