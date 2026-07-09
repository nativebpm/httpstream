package httpstream

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		wantErr bool
	}{
		{
			name:    "valid URL",
			baseURL: "https://example.com",
			wantErr: false,
		},
		{
			name:    "invalid URL",
			baseURL: "://invalid",
			wantErr: true,
		},
		{
			name:    "empty URL",
			baseURL: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{}
			_, err := NewClient(client, tt.baseURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewClient_NilClient(t *testing.T) {
	client, err := NewClient(nil, "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error with nil client: %v", err)
	}
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestClient_url(t *testing.T) {
	client := &http.Client{}
	hc, _ := NewClient(client, "https://example.com/api")

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "simple path",
			path: "/users",
			want: "https://example.com/api/users",
		},
		{
			name: "empty path",
			path: "",
			want: "https://example.com/api",
		},
		{
			name: "path with leading slash",
			path: "/posts/1",
			want: "https://example.com/api/posts/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hc.url(tt.path)
			if got != tt.want {
				t.Errorf("url() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_NewRequest(t *testing.T) {
	client := &http.Client{}
	hc, _ := NewClient(client, "https://example.com")

	ctx := context.Background()
	req := hc.Request(ctx, GET, "/test")

	if req.Method != "GET" {
		t.Errorf("NewRequest() method = %v, want GET", req.Method)
	}

	expectedURL := "https://example.com/test"
	if req.URL.String() != expectedURL {
		t.Errorf("NewRequest() URL = %v, want %v", req.URL.String(), expectedURL)
	}
}

func TestClient_NewMultipart(t *testing.T) {
	client := &http.Client{}
	hc, _ := NewClient(client, "https://example.com")

	ctx := context.Background()
	mp := hc.MultipartRequest(ctx, POST, "/upload")

	if mp == nil {
		t.Error("NewMultipart() should return a non-nil Multipart")
	}

	// Since request field is unexported, we can't check method/URL directly
	// But we can check that it's initialized
}

func TestClient_WithMiddleware(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test") != "middleware" {
			t.Errorf("Expected X-Test header to be set by middleware")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{}
	hc, _ := NewClient(client, server.URL)
	hc.Use(func(rt http.RoundTripper) http.RoundTripper {
		return &testTransport{rt: rt}
	})

	ctx := context.Background()
	req := hc.Request(ctx, GET, "/")

	resp, err := req.Send()
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// testTransport is a helper to add headers for testing middleware
type testTransport struct {
	rt http.RoundTripper
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("X-Test", "middleware")
	return t.rt.RoundTrip(req)
}

func TestRequest_Stream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("stream-data"))
	}))
	defer server.Close()

	hc, _ := NewClient(nil, server.URL)
	ctx := context.Background()
	req := hc.GET(ctx, "/")

	stream, err := req.Stream()
	if err != nil {
		t.Fatalf("Stream() failed: %v", err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("ReadAll() failed: %v", err)
	}

	if string(data) != "stream-data" {
		t.Errorf("Expected 'stream-data', got %q", string(data))
	}
}

func TestRequest_StreamLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("line1\nline2\n"))
	}))
	defer server.Close()

	hc, _ := NewClient(nil, server.URL)
	ctx := context.Background()
	req := hc.GET(ctx, "/")

	var lines []string
	err := req.StreamLines(func(line string) error {
		lines = append(lines, strings.TrimSuffix(line, "\n"))
		return nil
	})
	if err != nil {
		t.Fatalf("StreamLines() failed: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" {
		t.Errorf("Unexpected lines: %v", lines)
	}
}

func TestRequest_StreamSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		payload := ": comment\n" +
			"id: 101\n" +
			"event: message\n" +
			"data: line-one\n" +
			"data: line-two\n\n" +
			"id: 102\n" +
			"data: single-line\n\n"
		_, _ = w.Write([]byte(payload))
	}))
	defer server.Close()

	hc, _ := NewClient(nil, server.URL)
	ctx := context.Background()
	req := hc.GET(ctx, "/")

	var events []Event
	err := req.StreamSSE(func(event Event) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamSSE() failed: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	e1 := events[0]
	if e1.ID != "101" || e1.Event != "message" || e1.Data != "line-one\nline-two" {
		t.Errorf("Unexpected event 1: %+v", e1)
	}

	e2 := events[1]
	if e2.ID != "102" || e2.Event != "" || e2.Data != "single-line" {
		t.Errorf("Unexpected event 2: %+v", e2)
	}
}

func TestRequest_IdleTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("line1\n"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Wait 300ms before sending the next line to trigger the 100ms idle timeout
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte("line2\n"))
	}))
	defer server.Close()

	hc, _ := NewClient(nil, server.URL)
	ctx := context.Background()
	req := hc.GET(ctx, "/").IdleTimeout(100 * time.Millisecond)

	var lines []string
	err := req.StreamLines(func(line string) error {
		lines = append(lines, strings.TrimSuffix(line, "\n"))
		return nil
	})

	if !errors.Is(err, ErrStreamTimeout) {
		t.Errorf("Expected ErrStreamTimeout error, got: %v", err)
	}

	if len(lines) != 1 || lines[0] != "line1" {
		t.Errorf("Expected only 'line1' to be read before timeout, read: %v", lines)
	}
}
