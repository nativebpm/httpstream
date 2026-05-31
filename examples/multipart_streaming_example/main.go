package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"runtime"
	"time"

	"github.com/nativebpm/httpstream"
	// "github.com/nativebpm/httpstream/examples/multipart_streaming_example/middleware"
)

// countingReader wraps an io.Reader and tracks the number of bytes read
type countingReader struct {
	reader io.Reader
	count  int64
}

func (cr *countingReader) Read(p []byte) (n int, err error) {
	n, err = cr.reader.Read(p)
	cr.count += int64(n)
	return n, err
}

func (cr *countingReader) Close() error {
	if closer, ok := cr.reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func main() {
	logger := slog.Default()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logger.Info("Before streaming", "Alloc (KB)", m.Alloc/1024, "TotalAlloc (KB)", m.TotalAlloc/1024)

	client := &http.Client{Timeout: 60 * time.Second}

	server1Client, err := httpstream.NewClient(client, "http://localhost:8080")
	if err != nil {
		logger.Error("Failed to server1Client client", "error", err)
		return
	}

	server1Client.Use(httpstream.LoggingMiddleware(logger.WithGroup("server1")))
	// server1Client.Use(middleware.ProgressMiddleware(logger.WithGroup("server1")))

	server2Client, err := httpstream.NewClient(client, "http://localhost:8081")
	if err != nil {
		logger.Error("Failed to server2Client client", "error", err)
		return
	}

	server1Client.Use(httpstream.LoggingMiddleware(logger.WithGroup("server2")))
	// server1Client.Use(middleware.UploadProgressMiddleware(logger.WithGroup("server2")))

	server1Resp, err := server1Client.GET(context.Background(), "/file").
		Timeout(30 * time.Second).
		Send()
	if err != nil {
		logger.Error("Failed to get file from server1", "error", err)
		return
	}
	defer server1Resp.Body.Close()

	if server1Resp.StatusCode != http.StatusOK {
		logger.Error("Server1 returned status", "status", server1Resp.Status)
		return
	}

	filename := filename(server1Resp.Header, "default_filename")

	// Wrap response body with counting reader to track streamed data
	counter := &countingReader{reader: server1Resp.Body}

	server2Resp, err := server2Client.Multipart(context.Background(), "/upload").
		File("file", filename, counter).
		Timeout(30 * time.Second).
		Send()

	if err != nil {
		logger.Error("Failed to upload file", "error", err)
		return
	}
	defer server2Resp.Body.Close()

	if server2Resp.StatusCode != http.StatusOK {
		logger.Error("Upload failed with status", "status", server2Resp.Status)
		return
	}

	runtime.ReadMemStats(&m)
	slog.Info("After streaming", "Alloc (KB)", m.Alloc/1024, "TotalAlloc (KB)", m.TotalAlloc/1024)

	// Log the amount of data streamed
	streamedMB := float64(counter.count) / (1024 * 1024)
	slog.Info("Data streamed through pipeline",
		"bytes", counter.count,
		"megabytes", fmt.Sprintf("%.2f MB", streamedMB))

	body, err := io.ReadAll(server2Resp.Body)
	if err != nil {
		logger.Error("Failed to read response", "error", err)
		return
	}

	logger.Info("Upload successful", "server2Resp response", string(body))
}

func filename(headers http.Header, defaultName string) string {
	if v := headers.Get("Content-Disposition"); v != "" {
		_, params, err := mime.ParseMediaType(v)
		if err == nil {
			if fn, ok := params["filename"]; ok {
				return fn
			}
		}
	}
	return defaultName
}
