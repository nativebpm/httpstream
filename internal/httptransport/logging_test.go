package httptransport_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nativebpm/httpstream"
	"github.com/nativebpm/httpstream/internal/httptransport"
)

func TestLoggingMiddleware_EndToEnd(t *testing.T) {
	// test server that returns 200 OK
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-test", "ok")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	// create a JSON logger writing to io.Discard so tests don't print to stdout
	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))

	client, err := httpstream.NewClient(&http.Client{Timeout: 5 * time.Second}, ts.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	req := client.GET(context.Background(), "/").Use(httptransport.LoggingMiddleware(logger))
	resp, err := req.Send()
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}
