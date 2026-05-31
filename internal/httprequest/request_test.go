package httprequest_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nativebpm/httpstream/internal/httprequest"
)

func TestNewRequest(t *testing.T) {
	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse("http://example.com/api")

	req := httprequest.NewRequest(ctx, client, http.MethodGet, url.String())
	if req == nil {
		t.Fatal("NewRequest returned nil")
	}
}

func TestRequest_PathParam(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		params   map[string]string
		expected string
	}{
		{
			name:     "single_param",
			url:      "/users/{id}",
			params:   map[string]string{"id": "123"},
			expected: "/users/123",
		},
		{
			name:     "multiple_params",
			url:      "/users/{userId}/posts/{postId}",
			params:   map[string]string{"userId": "123", "postId": "456"},
			expected: "/users/123/posts/456",
		},
		{
			name:     "param_with_special_chars",
			url:      "/files/{filename}",
			params:   map[string]string{"filename": "my-file.pdf"},
			expected: "/files/my-file.pdf",
		},
		{
			name:     "param_at_end",
			url:      "/api/v1/resource/{id}",
			params:   map[string]string{"id": "abc-def-ghi"},
			expected: "/api/v1/resource/abc-def-ghi",
		},
		{
			name:     "multiple_same_param",
			url:      "/path/{param}/nested/{param}",
			params:   map[string]string{"param": "value"},
			expected: "/path/value/nested/value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := http.Client{}
			ctx := context.Background()

			url, _ := url.Parse(server.URL + tt.url)

			req := httprequest.NewRequest(ctx, client, http.MethodGet, url.String())
			for key, value := range tt.params {
				req = req.PathParam(key, value)
			}

			resp, err := req.Send()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if receivedPath != tt.expected {
				t.Errorf("expected path %s, got %s", tt.expected, receivedPath)
			}
		})
	}
}

func TestRequest_PathInt(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL + "/users/{id}/score/{score}")

	resp, err := httprequest.NewRequest(ctx, client, http.MethodGet,
		url.String()).
		PathInt("id", 123).
		PathInt("score", 95).
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	expected := "/users/123/score/95"
	if receivedPath != expected {
		t.Errorf("expected path %s, got %s", expected, receivedPath)
	}
}

func TestRequest_PathBool(t *testing.T) {
	tests := []struct {
		name     string
		value    bool
		expected string
	}{
		{
			name:     "true",
			value:    true,
			expected: "/api/active/true",
		},
		{
			name:     "false",
			value:    false,
			expected: "/api/active/false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := http.Client{}
			ctx := context.Background()

			url, _ := url.Parse(server.URL + "/api/active/{status}")

			resp, err := httprequest.NewRequest(ctx, client, http.MethodGet,
				url.String()).
				PathBool("status", tt.value).
				Send()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if receivedPath != tt.expected {
				t.Errorf("expected path %s, got %s", tt.expected, receivedPath)
			}
		})
	}
}

func TestRequest_PathFloat(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL + "/products/{price}")

	resp, err := httprequest.NewRequest(ctx, client, http.MethodGet,
		url.String()).
		PathFloat("price", 19.99).
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	expected := "/products/19.99"
	if receivedPath != expected {
		t.Errorf("expected path %s, got %s", expected, receivedPath)
	}
}

func TestRequest_PathParamWithQueryParams(t *testing.T) {
	var receivedPath string
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL + "/users/{id}/posts")

	resp, err := httprequest.NewRequest(ctx, client,
		http.MethodGet, url.String()).
		PathParam("id", "123").
		Param("page", "2").
		Param("limit", "10").
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	expectedPath := "/users/123/posts"
	if receivedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, receivedPath)
	}

	if !strings.Contains(receivedQuery, "page=2") {
		t.Errorf("expected query to contain page=2, got %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "limit=10") {
		t.Errorf("expected query to contain limit=10, got %s", receivedQuery)
	}
}

func TestRequest_PathParamChaining(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL + "/api/{version}/users/{id}")

	// Test that all methods return *Request for chaining
	req := httprequest.NewRequest(ctx, client,
		http.MethodGet, url.String()).
		PathParam("version", "v1").
		Header("X-Custom", "value").
		PathInt("id", 123).
		Param("filter", "active").
		Bool("verbose", true)

	resp, err := req.Send()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestRequest_Param(t *testing.T) {
	var receivedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL + "/api")

	resp, err := httprequest.NewRequest(ctx, client, http.MethodGet, url.String()).
		Param("key1", "value1").
		Param("key2", "value2").
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if !strings.Contains(receivedQuery, "key1=value1") {
		t.Errorf("expected query to contain key1=value1, got %s", receivedQuery)
	}
	if !strings.Contains(receivedQuery, "key2=value2") {
		t.Errorf("expected query to contain key2=value2, got %s", receivedQuery)
	}
}

func TestRequest_TypedParams(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*httprequest.Request) *httprequest.Request
		expected map[string]string
	}{
		{
			name: "bool_true",
			setup: func(r *httprequest.Request) *httprequest.Request {
				return r.Bool("active", true)
			},
			expected: map[string]string{"active": "true"},
		},
		{
			name: "bool_false",
			setup: func(r *httprequest.Request) *httprequest.Request {
				return r.Bool("active", false)
			},
			expected: map[string]string{"active": "false"},
		},
		{
			name: "int",
			setup: func(r *httprequest.Request) *httprequest.Request {
				return r.Int("count", 42)
			},
			expected: map[string]string{"count": "42"},
		},
		{
			name: "float",
			setup: func(r *httprequest.Request) *httprequest.Request {
				return r.Float("price", 19.99)
			},
			expected: map[string]string{"price": "19.99"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedQuery string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedQuery = r.URL.RawQuery
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := http.Client{}
			ctx := context.Background()

			url, _ := url.Parse(server.URL + "/api")

			req := httprequest.NewRequest(ctx, client, http.MethodGet, url.String())
			resp, err := tt.setup(req).Send()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			for key, expected := range tt.expected {
				expectedParam := key + "=" + expected
				if !strings.Contains(receivedQuery, expectedParam) {
					t.Errorf("expected query to contain %s, got %s", expectedParam, receivedQuery)
				}
			}
		})
	}
}

func TestRequest_Header(t *testing.T) {
	var receivedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL)

	resp, err := httprequest.NewRequest(ctx, client, http.MethodGet, url.String()).
		Header("X-API-Key", "secret-token-123").
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if receivedHeader != "secret-token-123" {
		t.Errorf("expected header X-API-Key=secret-token-123, got %s", receivedHeader)
	}
}

func TestRequest_JSON(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	var receivedUser User
	var receivedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &receivedUser); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL + "/api/users")

	user := User{Name: "John Doe", Email: "john@example.com"}
	resp, err := httprequest.NewRequest(ctx, client, http.MethodPost, url.String()).
		JSON(user).
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if receivedContentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", receivedContentType)
	}

	if receivedUser.Name != user.Name || receivedUser.Email != user.Email {
		t.Errorf("expected user %+v, got %+v", user, receivedUser)
	}
}

func TestRequest_Body(t *testing.T) {
	var receivedBody string
	var receivedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		receivedBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL)

	bodyContent := "custom body content"
	resp, err := httprequest.NewRequest(ctx, client, http.MethodPost, url.String()).
		Body(io.NopCloser(strings.NewReader(bodyContent)), "text/plain").
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if receivedContentType != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %s", receivedContentType)
	}

	if receivedBody != bodyContent {
		t.Errorf("expected body %s, got %s", bodyContent, receivedBody)
	}
}

func TestRequest_ContextCancellation(t *testing.T) {
	blockCh := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blockCh // Block until test cleanup
		w.WriteHeader(http.StatusOK)
	}))
	defer func() {
		close(blockCh)
		server.Close()
	}()

	client := http.Client{}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	url, _ := url.Parse(server.URL)

	_, err := httprequest.NewRequest(ctx, client, http.MethodGet, url.String()).Send()

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected context deadline exceeded error, got: %v", err)
	}
}

func TestRequest_Timeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}

	url, _ := url.Parse(server.URL)

	// Test 1: Request should timeout
	ctx := context.Background()
	_, err := httprequest.NewRequest(ctx, client, http.MethodGet, url.String()).
		Timeout(50 * time.Millisecond).
		Send()

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}

	url, _ = url.Parse(server.URL)

	// Test 2: Request should succeed with longer timeout
	resp, err := httprequest.NewRequest(ctx, client, http.MethodGet, url.String()).
		Timeout(500 * time.Millisecond).
		Send()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	}
}

func TestRequest_JSONWithTimeout(t *testing.T) {
	// Server that processes slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL)

	data := map[string]string{"key": "value"}
	_, err := httprequest.NewRequest(ctx, client, http.MethodPost, url.String()).
		JSON(data).
		Timeout(50 * time.Millisecond).
		Send()

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}
}

func TestRequest_ComplexChaining(t *testing.T) {
	type RequestData struct {
		Name   string  `json:"name"`
		Price  float64 `json:"price"`
		Active bool    `json:"active"`
	}

	var receivedPath string
	var receivedQuery string
	var receivedHeader string
	var receivedData RequestData

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedQuery = r.URL.RawQuery
		receivedHeader = r.Header.Get("Authorization")

		body, _ := io.ReadAll(r.Body)
		err := json.Unmarshal(body, &receivedData)
		if err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	url, _ := url.Parse(server.URL + "/api/{version}/products/{id}")

	data := RequestData{Name: "Product", Price: 99.99, Active: true}
	resp, err := httprequest.NewRequest(ctx, client, http.MethodPost, url.String()).
		PathParam("version", "v1").
		PathInt("id", 123).
		Header("Authorization", "Bearer token123").
		Param("source", "web").
		Bool("notify", true).
		JSON(data).
		Timeout(5 * time.Second).
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	expectedPath := "/api/v1/products/123"
	if receivedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, receivedPath)
	}

	if !strings.Contains(receivedQuery, "source=web") {
		t.Errorf("expected query to contain source=web, got %s", receivedQuery)
	}

	if receivedHeader != "Bearer token123" {
		t.Errorf("expected Authorization header, got %s", receivedHeader)
	}

	if receivedData.Name != data.Name {
		t.Errorf("expected data name %s, got %s", data.Name, receivedData.Name)
	}
}
