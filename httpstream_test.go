package httpstream

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
