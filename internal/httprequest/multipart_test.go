package httprequest_test

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nativebpm/httpstream/internal/httprequest"
)

func TestNewMultipart(t *testing.T) {
	client := http.Client{}
	ctx := context.Background()

	mp := httprequest.NewMultipart(ctx, client, http.MethodPost, "http://example.com/upload")
	if mp == nil {
		t.Fatal("NewMultipart returned nil")
	}
}

func TestMultipart_Param(t *testing.T) {
	receivedFields := make(map[string]string)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			t.Errorf("failed to parse media type: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mediaType != "multipart/form-data" {
			t.Errorf("expected multipart/form-data, got %s", mediaType)
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("failed to read part: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			data, err := io.ReadAll(p)
			if err != nil {
				t.Errorf("failed to read part data: %v", err)
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			receivedFields[p.FormName()] = string(data)
			p.Close()
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		Param("name", "John Doe").
		Param("email", "john@example.com").
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if receivedFields["name"] != "John Doe" {
		t.Errorf("expected name=John Doe, got %s", receivedFields["name"])
	}
	if receivedFields["email"] != "john@example.com" {
		t.Errorf("expected email=john@example.com, got %s", receivedFields["email"])
	}
}

func TestMultipart_TypedFields(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*httprequest.Multipart) *httprequest.Multipart
		expected map[string]string
	}{
		{
			name: "bool_true",
			setup: func(mp *httprequest.Multipart) *httprequest.Multipart {
				return mp.Bool("active", true)
			},
			expected: map[string]string{"active": "true"},
		},
		{
			name: "bool_false",
			setup: func(mp *httprequest.Multipart) *httprequest.Multipart {
				return mp.Bool("active", false)
			},
			expected: map[string]string{"active": "false"},
		},
		{
			name: "float",
			setup: func(mp *httprequest.Multipart) *httprequest.Multipart {
				return mp.Float("price", 19.99)
			},
			expected: map[string]string{"price": "19.99"},
		},
		{
			name: "int",
			setup: func(mp *httprequest.Multipart) *httprequest.Multipart {
				return mp.Int("count", 42)
			},
			expected: map[string]string{"count": "42"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receivedFields := make(map[string]string)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contentType := r.Header.Get("Content-Type")
				mediaType, params, err := mime.ParseMediaType(contentType)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				if mediaType != "multipart/form-data" {
					http.Error(w, "invalid content type", http.StatusBadRequest)
					return
				}

				mr := multipart.NewReader(r.Body, params["boundary"])
				for {
					p, err := mr.NextPart()
					if err == io.EOF {
						break
					}
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					data, err := io.ReadAll(p)
					if err != nil {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
					receivedFields[p.FormName()] = string(data)
					p.Close()
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			client := http.Client{}
			ctx := context.Background()

			mp := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL)
			resp, err := tt.setup(mp).Send()

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}

			for key, expected := range tt.expected {
				if receivedFields[key] != expected {
					t.Errorf("expected %s=%s, got %s", key, expected, receivedFields[key])
				}
			}
		})
	}
}

func TestMultipart_File(t *testing.T) {
	receivedFiles := make(map[string][]byte)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mediaType != "multipart/form-data" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			data, err := io.ReadAll(p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			receivedFiles[p.FormName()] = data
			p.Close()
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	fileContent := []byte("Hello, World!")
	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		File("document", "test.txt", bytes.NewReader(fileContent)).
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if !bytes.Equal(receivedFiles["document"], fileContent) {
		t.Errorf("expected file content %q, got %q", fileContent, receivedFiles["document"])
	}
}

func TestMultipart_MixedParamsAndFiles(t *testing.T) {
	receivedFields := make(map[string]string)
	receivedFiles := make(map[string][]byte)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mediaType != "multipart/form-data" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			data, err := io.ReadAll(p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			if p.FileName() != "" {
				receivedFiles[p.FormName()] = data
			} else {
				receivedFields[p.FormName()] = string(data)
			}
			p.Close()
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	fileContent := []byte("PDF content here")
	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		Param("title", "My Document").
		Int("version", 2).
		Bool("published", true).
		File("pdf", "document.pdf", bytes.NewReader(fileContent)).
		Param("author", "Jane Doe").
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	expectedFields := map[string]string{
		"title":     "My Document",
		"version":   "2",
		"published": "true",
		"author":    "Jane Doe",
	}

	for key, expected := range expectedFields {
		if receivedFields[key] != expected {
			t.Errorf("expected %s=%s, got %s", key, expected, receivedFields[key])
		}
	}

	if !bytes.Equal(receivedFiles["pdf"], fileContent) {
		t.Errorf("expected file content %q, got %q", fileContent, receivedFiles["pdf"])
	}
}

func TestMultipart_Header(t *testing.T) {
	var receivedHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-API-Key")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		Header("X-API-Key", "secret-token-123").
		Param("data", "value").
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if receivedHeader != "secret-token-123" {
		t.Errorf("expected header X-API-Key=secret-token-123, got %s", receivedHeader)
	}
}

func TestMultipart_LargeFile(t *testing.T) {
	receivedSize := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mediaType != "multipart/form-data" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			n, err := io.Copy(io.Discard, p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			receivedSize += int(n)
			p.Close()
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	// Create a 1MB file content
	largeContent := bytes.Repeat([]byte("x"), 1024*1024)
	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		File("largefile", "large.dat", bytes.NewReader(largeContent)).
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if receivedSize != len(largeContent) {
		t.Errorf("expected file size %d, got %d", len(largeContent), receivedSize)
	}
}

func TestMultipart_ContextCancellation(t *testing.T) {
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

	_, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		Param("data", "value").
		Send()

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}

	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("expected context deadline exceeded error, got: %v", err)
	}
}

func TestMultipart_MultipleFiles(t *testing.T) {
	receivedFiles := make(map[string][]byte)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mediaType != "multipart/form-data" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			data, err := io.ReadAll(p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			key := p.FormName() + ":" + p.FileName()
			receivedFiles[key] = data
			p.Close()
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	file1 := []byte("content1")
	file2 := []byte("content2")
	file3 := []byte("content3")

	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		File("file", "doc1.txt", bytes.NewReader(file1)).
		File("file", "doc2.txt", bytes.NewReader(file2)).
		File("attachment", "doc3.txt", bytes.NewReader(file3)).
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if !bytes.Equal(receivedFiles["file:doc1.txt"], file1) {
		t.Errorf("file1 content mismatch")
	}
	if !bytes.Equal(receivedFiles["file:doc2.txt"], file2) {
		t.Errorf("file2 content mismatch")
	}
	if !bytes.Equal(receivedFiles["attachment:doc3.txt"], file3) {
		t.Errorf("file3 content mismatch")
	}
}

func TestMultipart_EmptyForm(t *testing.T) {
	partCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mediaType != "multipart/form-data" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			_, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			partCount++
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if partCount != 0 {
		t.Errorf("expected 0 parts, got %d", partCount)
	}
}

func TestMultipart_ChainedCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	// Test that all methods return *Multipart for chaining
	mp := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		Header("X-Custom", "value").
		Param("key1", "value1").
		Bool("flag", true).
		Float("price", 9.99).
		Int("count", 5).
		File("doc", "file.txt", strings.NewReader("content")).
		Param("key2", "value2")

	resp, err := mp.Send()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestMultipart_Timeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}

	// Test 1: Request should timeout
	ctx := context.Background()
	_, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		Param("field", "value").
		Timeout(50 * time.Millisecond).
		Send()

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}

	// Test 2: Request should succeed with longer timeout
	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		Param("field", "value").
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

func TestMultipart_TimeoutChaining(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	// Test chaining Timeout with other methods
	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		Timeout(5*time.Second).
		Header("X-Test", "value").
		Param("field", "value").
		File("file", "test.txt", strings.NewReader("content")).
		Send()

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp != nil {
		resp.Body.Close()
	}
}

func TestMultipart_TimeoutWithLargeFile(t *testing.T) {
	// Server that reads slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read body slowly
		buf := make([]byte, 1024)
		for {
			_, err := r.Body.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				return
			}
			time.Sleep(10 * time.Millisecond) // Slow read
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	// Large file that should timeout during upload
	largeContent := bytes.Repeat([]byte("x"), 1024*100) // 100KB
	_, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
		File("largefile", "large.dat", bytes.NewReader(largeContent)).
		Timeout(50 * time.Millisecond).
		Send()

	if err == nil {
		t.Error("Expected timeout error for large file upload, got nil")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}
}

func TestMultipart_PathParam(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		params   map[string]string
		expected string
	}{
		{
			name:     "single_param",
			url:      "/users/{id}/avatar",
			params:   map[string]string{"id": "123"},
			expected: "/users/123/avatar",
		},
		{
			name:     "multiple_params",
			url:      "/users/{userId}/files/{fileId}",
			params:   map[string]string{"userId": "123", "fileId": "456"},
			expected: "/users/123/files/456",
		},
		{
			name:     "param_with_special_chars",
			url:      "/projects/{name}/upload",
			params:   map[string]string{"name": "my-project"},
			expected: "/projects/my-project/upload",
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

			mp := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL+tt.url)
			for key, value := range tt.params {
				mp = mp.PathParam(key, value)
			}

			resp, err := mp.Param("field", "value").Send()
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

func TestMultipart_PathInt(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost,
		server.URL+"/users/{id}/files/{version}",
	).
		PathInt("id", 123).
		PathInt("version", 2).
		Param("name", "document").
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	expected := "/users/123/files/2"
	if receivedPath != expected {
		t.Errorf("expected path %s, got %s", expected, receivedPath)
	}
}

func TestMultipart_PathBool(t *testing.T) {
	tests := []struct {
		name     string
		value    bool
		expected string
	}{
		{
			name:     "true",
			value:    true,
			expected: "/api/public/true/upload",
		},
		{
			name:     "false",
			value:    false,
			expected: "/api/public/false/upload",
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

			resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost,
				server.URL+"/api/public/{visibility}/upload",
			).
				PathBool("visibility", tt.value).
				Param("title", "File").
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

func TestMultipart_PathFloat(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost,
		server.URL+"/products/{price}/image",
	).
		PathFloat("price", 99.95).
		File("image", "product.jpg", strings.NewReader("image content")).
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	expected := "/products/99.95/image"
	if receivedPath != expected {
		t.Errorf("expected path %s, got %s", expected, receivedPath)
	}
}

func TestMultipart_PathParamWithFile(t *testing.T) {
	var receivedPath string
	receivedFiles := make(map[string][]byte)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path

		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if mediaType != "multipart/form-data" {
			http.Error(w, "invalid content type", http.StatusBadRequest)
			return
		}

		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			data, err := io.ReadAll(p)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			receivedFiles[p.FormName()] = data
			p.Close()
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	fileContent := []byte("document content")
	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost,
		server.URL+"/users/{userId}/documents/{docType}",
	).
		PathParam("userId", "abc-123").
		PathParam("docType", "invoice").
		File("document", "invoice.pdf", bytes.NewReader(fileContent)).
		Param("title", "Invoice 2025").
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	expectedPath := "/users/abc-123/documents/invoice"
	if receivedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, receivedPath)
	}

	if !bytes.Equal(receivedFiles["document"], fileContent) {
		t.Errorf("file content mismatch")
	}
}

func TestMultipart_ComplexPathChaining(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost,
		server.URL+"/api/{version}/users/{id}/files/{fileId}",
	).
		PathParam("version", "v2").
		Header("Authorization", "Bearer token").
		PathInt("id", 456).
		Param("description", "File upload").
		PathParam("fileId", "xyz-789").
		File("file", "document.pdf", strings.NewReader("content")).
		Timeout(5 * time.Second).
		Send()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	expectedPath := "/api/v2/users/456/files/xyz-789"
	if receivedPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, receivedPath)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}
