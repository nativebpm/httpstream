package httprequest

import (
	"context"
	"io"
)

type cancelCloser struct {
	io.ReadCloser
	cancelFunc context.CancelFunc
}

func (c *cancelCloser) Close() error {
	err := c.ReadCloser.Close()
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	return err
}

type contentType string

const (
	applicationOctetStream    contentType = "application/octet-stream"
	multipartFormData         contentType = "multipart/form-data"
	applicationJSON           contentType = "application/json"
	applicationUrlEncodedForm contentType = "application/x-www-form-urlencoded"
)
