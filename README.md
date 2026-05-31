# httpstream — Stream-first HTTP client for Go

Efficient, zero-buffer streaming for large HTTP payloads — built on top of `net/http`.

```bash
go get github.com/nativebpm/httpstream
```

## Overview

`httpstream` provides a minimal, streaming-oriented API for building HTTP requests without buffering entire payloads in memory.  
Ideal for large JSON bodies, multipart uploads, generated archives, or continuous data feeds.

### Key Features

- Stream data directly via `io.Pipe` — no intermediate buffers
- Constant memory usage (`O(1)`), regardless of payload size
- Natural backpressure (writes block when receiver is slow)
- Thin `net/http` wrapper — fully compatible
- Middleware support: `func(http.RoundTripper) http.RoundTripper`
- Fluent API for readability (`GET`, `POST`, `Multipart`, etc.)
- No goroutine leaks, no globals

## How It Works

`httpstream` connects your writer directly to the HTTP transport.  
Data is transmitted as it’s produced — the server can start processing immediately,  
without waiting for the full body to be buffered.

## Why Streaming Matters

Traditional HTTP clients buffer request bodies entirely before sending.  
For large or dynamically generated payloads, this leads to:

- High memory usage (`O(n)` where n = payload size)
- Slow transmission start (server waits for full upload)
- Out-of-memory errors in constrained environments

`httpstream` eliminates these issues by design.

## Examples

- [Streaming multipart upload](examples/multipart_streaming_example)
- [Without fluent API (for comparison)](examples/multipart_streaming_example/multipart_streaming_without_fluent_api)
- [Logger middleware](examples/logger_slog_example)

## License

MIT — see [`LICENSE`](../LICENSE).
