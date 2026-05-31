package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"runtime"
	"time"

	// "github.com/nativebpm/httpstream/examples/multipart_streaming_example/middleware"
	"github.com/nativebpm/httpstream/internal/httptransport"
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

	httpstream := &http.Client{Timeout: 60 * time.Second}

	server1Client := *httpstream
	server2Client := *httpstream

	// Attach logging + progress middleware to server1 (for download progress).
	transport1 := http.DefaultTransport
	transport1 = httptransport.LoggingMiddleware(logger.WithGroup("server1"))(transport1)
	// transport1 = middleware.ProgressMiddleware(logger.WithGroup("server1"))(transport1)
	server1Client.Transport = transport1

	// Attach logging + upload-progress middleware to server2 (for upload progress).
	transport2 := http.DefaultTransport
	transport2 = httptransport.LoggingMiddleware(logger.WithGroup("server2"))(transport2)
	// transport2 = middleware.UploadProgressMiddleware(logger.WithGroup("server2"))(transport2)
	server2Client.Transport = transport2

	// GET /file from server1
	ctx1, cancel1 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel1()
	req1, err := http.NewRequestWithContext(ctx1, http.MethodGet, "http://localhost:8080/file", nil)
	if err != nil {
		logger.Error("Failed to create GET request", "error", err)
		return
	}
	server1Resp, err := server1Client.Do(req1)
	if err != nil {
		logger.Error("Failed to get file from server1", "error", err)
		return
	}
	defer server1Resp.Body.Close()

	if server1Resp.StatusCode != http.StatusOK {
		logger.Error("Server1 returned status", "status", server1Resp.Status)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Wrap response body with counting reader to track streamed data
	counter := &countingReader{reader: server1Resp.Body}

	// Streaming multipart upload to server2
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer pw.Close()
		defer mw.Close()

		filename := filename(server1Resp.Header, "default_filename")
		part, err := mw.CreateFormFile("file", filename)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		_, err = io.Copy(part, counter)
		if err != nil {
			pw.CloseWithError(err)
		}
	}()

	req2, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://localhost:8081/upload", pr)
	if err != nil {
		logger.Error("Failed to create POST request", "error", err)
		return
	}
	req2.Header.Set("Content-Type", mw.FormDataContentType())

	server2Resp, err := server2Client.Do(req2)
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
	logger.Info("After streaming", "Alloc (KB)", m.Alloc/1024, "TotalAlloc (KB)", m.TotalAlloc/1024)

	// Log the amount of data streamed
	streamedMB := float64(counter.count) / (1024 * 1024)
	logger.Info("Data streamed through pipeline",
		"bytes", counter.count,
		"megabytes", fmt.Sprintf("%.2f MB", streamedMB))

	body, err := io.ReadAll(server2Resp.Body)
	if err != nil {
		logger.Error("Failed to read response", "error", err)
		return
	}

	logger.Info("Upload successful", "server2Resp response", string(body))
}

// filename extracts filename from Content-Disposition header.
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
