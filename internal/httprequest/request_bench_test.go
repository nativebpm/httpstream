package httprequest_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nativebpm/httpstream/internal/httprequest"
)

func BenchmarkRequest_Simple(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	url, _ := url.Parse(server.URL)

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewRequest(ctx, client, http.MethodGet, url.String()).
			Header("X-API-Key", "secret").
			Param("page", "1").
			Send()
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkRequest_ManyParams(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	url, _ := url.Parse(server.URL)

	for i := 0; i < b.N; i++ {
		req := httprequest.NewRequest(ctx, client, http.MethodGet, url.String())
		for j := 0; j < 10; j++ {
			req.Param("param", "value")
		}
		resp, err := req.Send()
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkRequest_JSON(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	data := map[string]any{
		"name":  "John Doe",
		"age":   30,
		"email": "john@example.com",
	}

	b.ResetTimer()
	b.ReportAllocs()

	url, _ := url.Parse(server.URL)

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewRequest(ctx, client, http.MethodPost, url.String()).
			JSON(data).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkRequest_WithTimeout(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	url, _ := url.Parse(server.URL)

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewRequest(ctx, client, http.MethodGet, url.String()).
			Timeout(5 * time.Second).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkRequest_JSONWithTimeout(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	data := map[string]any{
		"name":  "John Doe",
		"age":   30,
		"email": "john@example.com",
	}

	b.ResetTimer()
	b.ReportAllocs()

	url, _ := url.Parse(server.URL)

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewRequest(ctx, client, http.MethodPost, url.String()).
			Timeout(10 * time.Second).
			JSON(data).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}

func BenchmarkRequest_ComplexChainWithTimeout(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	data := map[string]string{"key": "value"}

	b.ResetTimer()
	b.ReportAllocs()

	url, _ := url.Parse(server.URL)

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewRequest(ctx, client, http.MethodPost, url.String()).
			Header("X-API-Key", "secret").
			Header("User-Agent", "test-client").
			Timeout(5*time.Second).
			Param("page", "1").
			Param("limit", "10").
			JSON(data).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
