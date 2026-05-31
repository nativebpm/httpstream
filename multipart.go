package httpstream

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// multipartField represents a field in a multipart form
type multipartField struct {
	contentType contentType
	key, value  string
	file        io.Reader
}

// Multipart provides a streaming multipart/form-data builder for HTTP requests.
type Multipart struct {
	client     http.Client
	request    *http.Request
	fields     []multipartField
	cancelFunc context.CancelFunc
}

// NewMultipart creates a new streaming multipart/form-data request builder.
func NewMultipart(ctx context.Context, client http.Client, method, url string) *Multipart {
	request, _ := http.NewRequestWithContext(ctx, method, url, nil)
	return &Multipart{
		client:  client,
		request: request,
		fields:  make([]multipartField, 0, 16),
	}
}

func (r *Multipart) Use(middleware func(http.RoundTripper) http.RoundTripper) *Multipart {
	if r.client.Transport == nil {
		r.client.Transport = http.DefaultTransport
	}
	r.client.Transport = middleware(r.client.Transport)
	return r
}

// Timeout sets a timeout for the request.
func (r *Multipart) Timeout(duration time.Duration) *Multipart {
	ctx, cancel := context.WithTimeout(r.request.Context(), duration)
	r.cancelFunc = cancel
	r.request = r.request.WithContext(ctx)
	return r
}

// Send executes the HTTP request and returns the response.
func (r *Multipart) Send() (*http.Response, error) {
	ctx := r.request.Context()

	// Pre-validate files to ensure they are not empty
	for i, f := range r.fields {
		if f.contentType == applicationOctetStream {
			buf := make([]byte, 1)
			n, err := f.file.Read(buf)
			if err != nil && err != io.EOF {
				return nil, fmt.Errorf("failed to read file %s: %w", f.value, err)
			}
			if n == 0 {
				return nil, fmt.Errorf("empty file: %s", f.value)
			}
			r.fields[i].file = io.MultiReader(strings.NewReader(string(buf[:n])), f.file)
		}
	}

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	r.request.Body = pr
	r.request.Header.Set("Content-Type", mw.FormDataContentType())

	go func() {
		defer pw.Close()
		defer mw.Close()

		for _, f := range r.fields {
			select {
			case <-ctx.Done():
				pw.CloseWithError(ctx.Err())
				return
			default:
			}

			var err error
			if f.contentType == multipartFormData {
				err = mw.WriteField(f.key, f.value)
			} else if f.contentType == applicationOctetStream {
				var part io.Writer
				if part, err = mw.CreateFormFile(f.key, f.value); err == nil {
					_, err = io.Copy(part, f.file)
				}
			}
			if err != nil {
				pw.CloseWithError(err)
				return
			}
		}
	}()

	return r.sendRequest()
}

func (r *Multipart) sendRequest() (*http.Response, error) {
	resp, err := r.client.Do(r.request)
	if err != nil {
		if r.cancelFunc != nil {
			r.cancelFunc()
		}
		return nil, err
	}
	if r.cancelFunc != nil {
		resp.Body = &cancelCloser{resp.Body, r.cancelFunc}
		r.cancelFunc = nil
	}
	return resp, nil
}

// Header sets an HTTP header on the request.
func (r *Multipart) Header(k, v string) *Multipart { r.request.Header.Set(k, v); return r }

// PathParam replaces a path variable placeholder in the URL.
func (r *Multipart) PathParam(key, val string) *Multipart {
	r.request.URL.Path = strings.ReplaceAll(r.request.URL.Path, "{"+key+"}", val)
	return r
}
func (r *Multipart) PathInt(k string, v int) *Multipart     { return r.PathParam(k, strconv.Itoa(v)) }
func (r *Multipart) PathBool(k string, v bool) *Multipart   { return r.PathParam(k, strconv.FormatBool(v)) }
func (r *Multipart) PathFloat(k string, v float64) *Multipart { return r.PathParam(k, strconv.FormatFloat(v, 'f', -1, 64)) }

// Param adds a string field to the multipart form.
func (r *Multipart) Param(k, v string) *Multipart {
	r.fields = append(r.fields, multipartField{contentType: multipartFormData, key: k, value: v})
	return r
}
func (r *Multipart) Int(k string, v int) *Multipart     { return r.Param(k, strconv.Itoa(v)) }
func (r *Multipart) Bool(k string, v bool) *Multipart   { return r.Param(k, strconv.FormatBool(v)) }
func (r *Multipart) Float(k string, v float64) *Multipart { return r.Param(k, strconv.FormatFloat(v, 'f', -1, 64)) }

// File adds a file field to the multipart form.
func (r *Multipart) File(key, filename string, content io.Reader) *Multipart {
	r.fields = append(r.fields, multipartField{contentType: applicationOctetStream, key: key, value: filename, file: content})
	return r
}

// Cookie adds a cookie to the multipart request.
func (r *Multipart) Cookie(name, value string) *Multipart {
	r.request.AddCookie(&http.Cookie{Name: name, Value: value})
	return r
}
