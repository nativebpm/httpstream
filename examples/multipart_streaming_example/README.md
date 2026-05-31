# Multipart Streaming Example

This example demonstrates streaming data between two servers using httpstream.

## Architecture

- **Server 1** (:8080): Generates a large file and returns it upon request to `/file`.
- **Server 2** (:8081): Accepts a multipart request on `/upload` and save file.
- **Client**: Requests the file from Server 1, streams it directly into a multipart request to Server 2.

## Running

Run the servers and client in separate terminals:

1. **Server 1**:
```sh
go run server1/main.go
```
Server 1 running on :8080

2. **Server 2**:
```sh
go run server2/main.go
```
Server 2 running on :8081

3. **Client** (after starting the servers):
```sh
go run main.go
```
2025/10/06 09:22:41 INFO Before streaming "Alloc (KB)"=218 "TotalAlloc (KB)"=218
2025/10/06 09:22:41 INFO Sending request server1.method=GET server1.url=http://localhost:8080/file
2025/10/06 09:22:43 INFO Response received server1.status=200 server1.duration=1.849253692s
2025/10/06 09:22:43 INFO Sending request server2.method=POST server2.url=http://localhost:8081/upload
2025/10/06 09:22:47 INFO Response received server2.status=200 server2.duration=4.204263582s
2025/10/06 09:22:47 INFO After streaming "Alloc (KB)"=694 "TotalAlloc (KB)"=694
2025/10/06 09:22:47 INFO Data streamed through pipeline bytes=478888897 megabytes="456.70 MB"
2025/10/06 09:22:47 INFO Upload successful "server2Resp response"="File large.txt uploaded and save"

### Alternative: Without Fluent API

For comparison, there's an alternative implementation in `multipart_streaming_without_fluent_api/main.go` that achieves the same streaming behavior using only standard Go libraries (`net/http`, `mime/multipart`, `io.Pipe`).

To run the alternative client:
```sh
cd multipart_streaming_without_fluent_api
go run main.go
```

This version demonstrates how to implement multipart streaming manually, providing the same memory-efficient results.

## Result

- Server 1 generates file with numbered lines.
- Client streams the file from Server 1 to Server 2 without intermediate storage.
- Server 2 saves the file in its directory.
- Client outputs upload confirmation.
- Streaming progress in the logs

## Notes

- Data is transferred in a streaming manner, without loading into the client's memory.
- Uses `io.Reader` for streaming.
- In a real application, replace file generation with reading from a source.