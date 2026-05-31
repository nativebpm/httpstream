package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/nativebpm/streamhttp"
)

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	loggingMiddleware := streamhttp.LoggingMiddleware(logger)

	client, err := streamhttp.NewClient(http.Client{Timeout: 10 * time.Second}, "https://httpbin.org", loggingMiddleware)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.GET(context.Background(), "/get").Send()
	if err != nil {
		log.Fatal(err)
	}
	resp.Body.Close()
}
