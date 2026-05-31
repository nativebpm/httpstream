package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nativebpm/httpstream"
)

// BenchmarkStreamingUpload benchmarks the streaming version
func BenchmarkStreamingUpload(b *testing.B) {
	// Mock servers
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Generate large file (e.g., 10 MB of repeated data)
		data := bytes.Repeat([]byte("a"), 10*1024*1024)
		_, err := w.Write(data)
		if err != nil {
			b.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(io.Discard, r.Body) // Discard body like a real server would
		if err != nil {
			b.Errorf("Failed to read request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	client1, err := httpstream.NewClient(client, server1.URL)
	if err != nil {
		b.Fatal(err)
	}
	client2, err := httpstream.NewClient(client, server2.URL)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer() // Reset timer before the loop
	for i := 0; i < b.N; i++ {
		resp1, err := client1.GET(context.Background(), "/").Send()
		if err != nil {
			b.Fatal(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		resp2, err := client2.Multipart(ctx, "/upload").
			File("file", "test.txt", resp1.Body).
			Send()
		if err != nil {
			cancel()
			resp1.Body.Close()
			b.Fatal(err)
		}

		_, err = io.Copy(io.Discard, resp2.Body)
		if err != nil {
			b.Error(err)
		}

		// Close resources
		resp1.Body.Close()
		resp2.Body.Close()
		cancel()
	}
}

// BenchmarkBufferedUpload benchmarks the non-streaming version (buffering entire file)
func BenchmarkBufferedUpload(b *testing.B) {
	// Same mock server
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		data := bytes.Repeat([]byte("a"), 10*1024*1024)
		_, err := w.Write(data)
		if err != nil {
			b.Errorf("Failed to write response: %v", err)
		}
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			b.Errorf("Failed to read request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	client := &http.Client{Timeout: 30 * time.Second}
	client1, err := httpstream.NewClient(client, server1.URL)
	if err != nil {
		b.Fatal(err)
	}
	client2, err := httpstream.NewClient(client, server2.URL)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp1, err := client1.GET(context.Background(), "/").Send()
		if err != nil {
			b.Fatal(err)
		}

		// Buffer entire file in memory
		data, err := io.ReadAll(resp1.Body)
		if err != nil {
			resp1.Body.Close()
			b.Fatal(err)
		}
		resp1.Body.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		resp2, err := client2.Multipart(ctx, "/upload").
			File("file", "test.txt", bytes.NewReader(data)). // Use buffer
			Send()
		if err != nil {
			cancel()
			b.Fatal(err)
		}

		_, err = io.Copy(io.Discard, resp2.Body)
		if err != nil {
			b.Error(err)
		}
		resp2.Body.Close()
		cancel()
	}
}
