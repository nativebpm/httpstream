# Stream‑first HTTP client for Go.

Stream large payloads (without buffering) via `io.Pipe`.

```bash
go get github.com/nativebpm/httpclient
```

## Problem

Standard HTTP helpers buffer entire payloads in memory before sending. For large JSON, multipart uploads, generated archives, or database dumps this causes:
- High memory usage: O(n), where n = payload size
- Delayed transmission (server waits for full upload)
- OOM on constrained environments

## Solution

Stream data as you produce it:
- `io.Pipe` connects writer → HTTP transport directly
- Memory footprint: O(1) — constant, independent of payload size
- Natural backpressure via blocking writes
- Server can start processing immediately

## Features

- Thin `net/http` wrapper
- Middleware: `func(http.RoundTripper) http.RoundTripper`
- Fluent API (GET, POST, Multipart, etc.)
- No goroutine leaks, no globals

## Examples

- [Streaming multipart](examples/multipart_streaming_example)
- [Without fluent API (for code readability comparison)](examples/multipart_streaming_example/multipart_straming_without_fluent_api)
- [Logger middleware](examples/logger_slog_example)

## License

MIT — see [`LICENSE`](../LICENSE).