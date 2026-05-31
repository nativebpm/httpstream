package middleware

import (
	"io"
	"log/slog"
	"net/http"
)

type progressReader struct {
	reader io.ReadCloser
	logger *slog.Logger
}

func newProgressReader(reader io.ReadCloser, logger *slog.Logger) *progressReader {
	return &progressReader{reader: reader, logger: logger}
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		content := string(p[:n])
		if len(content) > 60 {
			content = content[:30] + "..." + content[len(content)-30:]
		}
		pr.logger.Info("Streaming", "content", content)
	}
	return n, err
}

func (pr *progressReader) Close() error {
	return pr.reader.Close()
}

func ProgressMiddleware(logger *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return progressRoundTripper{next: next, logger: logger}
	}
}

type progressRoundTripper struct {
	next   http.RoundTripper
	logger *slog.Logger
}

func (prt progressRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := prt.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.Body != nil {
		resp.Body = newProgressReader(resp.Body, prt.logger)
	}
	return resp, err
}

func UploadProgressMiddleware(logger *slog.Logger) func(http.RoundTripper) http.RoundTripper {
	return func(next http.RoundTripper) http.RoundTripper {
		return uploadProgressRoundTripper{next: next, logger: logger}
	}
}

type uploadProgressRoundTripper struct {
	next   http.RoundTripper
	logger *slog.Logger
}

func (uprt uploadProgressRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		req.Body = newProgressReader(req.Body, uprt.logger)
	}
	resp, err := uprt.next.RoundTrip(req)
	return resp, err
}
