package httpstream

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
)

// Stream executes the HTTP request and returns a raw readable response stream (io.ReadCloser).
func (r *Request) Stream() (io.ReadCloser, error) {
	resp, err := r.Send()
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("httpstream status non-OK: %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// StreamLines executes the HTTP request and invokes the callback sequentially for each line.
// This is suitable for processing custom line-delimited HTTP streams.
func (r *Request) StreamLines(callback func(line string) error) error {
	resp, err := r.Stream()
	if err != nil {
		return err
	}
	defer resp.Close()

	reader := bufio.NewReader(resp)
	for {
		select {
		case <-r.Context().Done():
			return r.Context().Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if line != "" {
					if cbErr := callback(line); cbErr != nil {
						return cbErr
					}
				}
				break
			}
			return err
		}

		if cbErr := callback(line); cbErr != nil {
			return cbErr
		}
	}
	return nil
}

// StreamSSE sets the "Accept: text/event-stream" header and processes the Server-Sent Events (SSE) stream.
// It executes the request and calls the callback for each streamed event line sequentially.
func (r *Request) StreamSSE(callback func(line string) error) error {
	r.Header("Accept", "text/event-stream")
	return r.StreamLines(callback)
}
