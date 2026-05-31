# Stream-first HTTP client for Go

A minimal, middleware-friendly HTTP client that streams data directly via `io.Pipe`, avoiding full in-memory buffering.

```bash
go get github.com/nativebpm/streamhttp
```

## Why

The standard `net/http` helpers buffer full payloads before sending. This becomes inefficient when working with:

- Large JSON or file uploads
- Generated archives or database exports
- Streaming pipelines and data transformations

## Problems

- **High memory usage** - O(n), proportional to payload size
- **Transmission delay** - server waits for full upload
- **OOM risk** - on constrained environments

## Stream-first solution

- `io.Pipe` connects your writer directly to the HTTP transport
- **Constant memory** - O(1), independent of payload size
- **Natural backpressure** - writer blocks until data is consumed by the transport
- **Immediate server processing** - no full buffering required

## Highlights

- Thin wrapper over `net/http`
- Composable middleware: `func(http.RoundTripper) http.RoundTripper`
- Fluent API: `GET`, `POST`, `Multipart`, etc.
- Safe for concurrent use - no globals, no internal state, no reflection

## Examples

- [Streaming multipart upload](examples/multipart_streaming_example)
- [Without fluent API (for comparison)](examples/multipart_streaming_example/multipart_streaming_without_fluent_api)
- [Logger middleware](examples/logger_slog_example)

## License

MIT - see [LICENSE](../LICENSE)
