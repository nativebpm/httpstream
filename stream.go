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
func (r *Request) StreamLines(callback func(line string) error) error {
	resp, err := r.Send()
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("httpstream status non-OK: %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
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
