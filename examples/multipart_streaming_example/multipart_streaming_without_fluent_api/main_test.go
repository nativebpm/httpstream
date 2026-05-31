package main

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// BenchmarkStreamingUploadStandard benchmarks streaming version with standard http.Client
func BenchmarkStreamingUploadStandard(b *testing.B) {
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
		_, err := io.Copy(io.Discard, r.Body) // Discard body like a real server
		if err != nil {
			b.Errorf("Failed to read request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	b.ResetTimer() // Reset timer before loop
	for i := 0; i < b.N; i++ {
		// GET
		req1, err := http.NewRequest("GET", server1.URL+"/", nil)
		if err != nil {
			b.Fatal(err)
		}
		resp1, err := client.Do(req1)
		if err != nil {
			b.Fatal(err)
		}

		// Multipart upload
		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)

		go func() {
			defer pw.Close()
			defer mw.Close()

			part, err := mw.CreateFormFile("file", "test.txt")
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			_, err = io.Copy(part, resp1.Body)
			if err != nil {
				pw.CloseWithError(err)
			}
		}()

		req2, err := http.NewRequestWithContext(context.Background(), "POST", server2.URL+"/upload", pr)
		if err != nil {
			b.Fatal(err)
		}
		req2.Header.Set("Content-Type", mw.FormDataContentType())

		resp2, err := client.Do(req2)
		if err != nil {
			b.Fatal(err)
		}

		_, err = io.Copy(io.Discard, resp2.Body)
		if err != nil {
			b.Error(err)
		}

		// Close resources
		resp1.Body.Close()
		resp2.Body.Close()
	}
}

// BenchmarkBufferedUploadStandard benchmarks non-streaming version (full file buffering)
func BenchmarkBufferedUploadStandard(b *testing.B) {
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// GET
		req1, err := http.NewRequest("GET", server1.URL+"/", nil)
		if err != nil {
			b.Fatal(err)
		}
		resp1, err := client.Do(req1)
		if err != nil {
			b.Fatal(err)
		}

		// Buffer entire file in memory
		data, err := io.ReadAll(resp1.Body)
		if err != nil {
			b.Fatal(err)
		}
		resp1.Body.Close()

		// Multipart upload
		pr, pw := io.Pipe()
		mw := multipart.NewWriter(pw)

		go func() {
			defer pw.Close()
			defer mw.Close()

			part, err := mw.CreateFormFile("file", "test.txt")
			if err != nil {
				pw.CloseWithError(err)
				return
			}
			_, err = io.Copy(part, bytes.NewReader(data))
			if err != nil {
				pw.CloseWithError(err)
			}
		}()

		req2, err := http.NewRequestWithContext(context.Background(), "POST", server2.URL+"/upload", pr)
		if err != nil {
			b.Fatal(err)
		}
		req2.Header.Set("Content-Type", mw.FormDataContentType())

		resp2, err := client.Do(req2)
		if err != nil {
			b.Fatal(err)
		}

		_, err = io.Copy(io.Discard, resp2.Body)
		if err != nil {
			b.Error(err)
		}
		resp2.Body.Close()
	}
}
