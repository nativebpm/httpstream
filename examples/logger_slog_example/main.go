package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/nativebpm/httpstream"
)

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	loggingMiddleware := httpstream.LoggingMiddleware(logger)

	client, err := httpstream.NewClient(&http.Client{Timeout: 10 * time.Second}, "https://httpbin.org")
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.GET(context.Background(), "/get").
		Use(loggingMiddleware).
		Send()
	if err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()
}
