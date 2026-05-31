package httprequest_test

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nativebpm/httpstream/internal/httprequest"
)

func BenchmarkMultipart_Simple(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			Param("name", "John Doe").
			Param("email", "john@example.com").
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkMultipart_ManyParams(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mp := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL)
		for j := 0; j < 20; j++ {
			mp.Param("key", "value")
		}
		resp, err := mp.Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkMultipart_TypedFields(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			Param("name", "Product").
			Int("quantity", 42).
			Float("price", 19.99).
			Bool("available", true).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkMultipart_SingleFile(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()
	fileContent := bytes.Repeat([]byte("x"), 1024) // 1KB file

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			File("document", "test.txt", bytes.NewReader(fileContent)).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkMultipart_SmallFile(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()
	fileContent := []byte("Hello, World!") // ~13 bytes

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			File("document", "small.txt", bytes.NewReader(fileContent)).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkMultipart_LargeFile(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()
	fileContent := bytes.Repeat([]byte("x"), 1024*1024) // 1MB file

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			File("document", "large.dat", bytes.NewReader(fileContent)).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkMultipart_MultipleFiles(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()
	file1 := bytes.Repeat([]byte("a"), 512)
	file2 := bytes.Repeat([]byte("b"), 512)
	file3 := bytes.Repeat([]byte("c"), 512)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			File("file1", "doc1.txt", bytes.NewReader(file1)).
			File("file2", "doc2.txt", bytes.NewReader(file2)).
			File("file3", "doc3.txt", bytes.NewReader(file3)).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkMultipart_MixedParamsAndFiles(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()
	fileContent := bytes.Repeat([]byte("x"), 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			Param("title", "Document").
			Param("author", "Jane Doe").
			Int("version", 2).
			Bool("published", true).
			Float("rating", 4.5).
			File("document", "doc.pdf", bytes.NewReader(fileContent)).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkMultipart_WithHeaders(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			Header("X-API-Key", "secret-token").
			Header("X-Request-ID", "req-123").
			Param("data", "value").
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

func BenchmarkMultipart_ChainedCalls(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()
	fileContent := []byte("content")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			Header("Authorization", "Bearer token").
			Param("param1", "value1").
			Param("param2", "value2").
			Int("count", 10).
			Bool("flag", true).
			Float("ratio", 0.75).
			File("attachment", "file.txt", bytes.NewReader(fileContent)).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// Benchmark for measuring server-side parsing overhead
func BenchmarkMultipart_ServerParsing(b *testing.B) {
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
			io.Copy(io.Discard, p)
			p.Close()
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	ctx := context.Background()
	fileContent := bytes.Repeat([]byte("x"), 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
			Param("field1", "value1").
			Param("field2", "value2").
			File("file", "data.bin", bytes.NewReader(fileContent)).
			Send()
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// Parallel benchmark to test concurrent multipart requests
func BenchmarkMultipart_Parallel(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := http.Client{}
	fileContent := bytes.Repeat([]byte("x"), 1024)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			resp, err := httprequest.NewMultipart(ctx, client, http.MethodPost, server.URL).
				Param("name", "test").
				File("document", "file.dat", bytes.NewReader(fileContent)).
				Send()
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}
