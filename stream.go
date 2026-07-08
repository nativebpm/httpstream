package httpstream

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Event represents a parsed Server-Sent Event (SSE).
type Event struct {
	ID    string
	Event string
	Data  string
}

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
// This is suitable for reading custom line-delimited raw text streams (like logs or stdout).
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

// StreamSSE sets the "Accept: text/event-stream" header, executes the request,
// and parses the response stream adhering to the W3C Server-Sent Events specification.
// It invokes the callback only when a complete event block is dispatched (on double newline).
func (r *Request) StreamSSE(callback func(event Event) error) error {
	r.Header("Accept", "text/event-stream")
	resp, err := r.Stream()
	if err != nil {
		return err
	}
	defer resp.Close()

	reader := bufio.NewReader(resp)
	var currentEvent Event
	var dataBuilder strings.Builder
	hasData := false

	for {
		select {
		case <-r.Context().Done():
			return r.Context().Err()
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		// Normalize line ending by stripping \r and \n
		trimmedLine := strings.TrimRight(line, "\r\n")

		if trimmedLine == "" {
			// A blank line dispatches the currently accumulated event if we have active content
			if hasData || currentEvent.Event != "" || currentEvent.ID != "" {
				currentEvent.Data = dataBuilder.String()
				if cbErr := callback(currentEvent); cbErr != nil {
					return cbErr
				}
				// Reset event buffer
				currentEvent = Event{}
				dataBuilder.Reset()
				hasData = false
			}
			if err == io.EOF {
				break
			}
			continue
		}

		// Comments (lines starting with colon) are ignored
		if strings.HasPrefix(trimmedLine, ":") {
			if err == io.EOF {
				break
			}
			continue
		}

		// Split line into field name and value
		var field, value string
		colonIdx := strings.Index(trimmedLine, ":")
		if colonIdx == -1 {
			field = trimmedLine
			value = ""
		} else {
			field = trimmedLine[:colonIdx]
			value = trimmedLine[colonIdx+1:]
			// Strip leading space if present
			if len(value) > 0 && value[0] == ' ' {
				value = value[1:]
			}
		}

		switch field {
		case "event":
			currentEvent.Event = value
		case "data":
			if hasData {
				dataBuilder.WriteByte('\n')
			}
			dataBuilder.WriteString(value)
			hasData = true
		case "id":
			currentEvent.ID = value
		case "retry":
			// We do not implement reconnection time changes, ignore
		}

		if err == io.EOF {
			// Dispatch any remaining event on stream termination
			if hasData || currentEvent.Event != "" || currentEvent.ID != "" {
				currentEvent.Data = dataBuilder.String()
				_ = callback(currentEvent)
			}
			break
		}
	}

	return nil
}
