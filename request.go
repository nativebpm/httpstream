package httpstream

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// requestPayload represents the body payload for standard HTTP requests
type requestPayload struct {
	contentType contentType
	content     any
	form        url.Values
}

// Request provides a builder for standard HTTP requests.
type Request struct {
	*http.Request
	client      http.Client
	body        requestPayload
	cancelFunc  context.CancelFunc
	gzip        bool // Gzip compression flag
	idleTimeout time.Duration
}

// NewRequest creates a new HTTP request builder.
func NewRequest(ctx context.Context, client http.Client, method string, url string) *Request {
	request, _ := http.NewRequestWithContext(ctx, method, url, nil)
	return &Request{
		Request: request,
		client:  client,
	}
}

// Gzip enables gzip compression for the request body stream on the fly.
func (r *Request) Gzip() *Request {
	r.gzip = true
	return r
}

func (r *Request) Use(middleware func(http.RoundTripper) http.RoundTripper) *Request {
	if r.client.Transport == nil {
		r.client.Transport = http.DefaultTransport
	}
	r.client.Transport = middleware(r.client.Transport)
	return r
}

// Timeout sets a timeout for the request.
func (r *Request) Timeout(duration time.Duration) *Request {
	ctx, cancel := context.WithTimeout(r.Context(), duration)
	r.cancelFunc = cancel
	r.Request = r.WithContext(ctx)
	return r
}

// IdleTimeout sets the maximum duration to wait for data between reads.
func (r *Request) IdleTimeout(duration time.Duration) *Request {
	r.idleTimeout = duration
	return r
}

// Send executes the HTTP request and returns the response.
func (r *Request) Send() (*http.Response, error) {
	ctx := r.Context()

	switch r.body.contentType {
	case applicationJSON:
		if r.body.content != nil {
			pr, pw := io.Pipe()
			r.Request.Body = pr

			go func() {
				defer pw.Close()

				select {
				case <-ctx.Done():
					pw.CloseWithError(ctx.Err())
					return
				default:
				}

				encoder := json.NewEncoder(pw)
				if err := encoder.Encode(r.body.content); err != nil {
					pw.CloseWithError(err)
					return
				}
			}()
		}
	case applicationUrlEncodedForm:
		if r.body.form != nil {
			r.Request.Body = io.NopCloser(strings.NewReader(r.body.form.Encode()))
		}
	}

	// Apply dynamic gzip stream pipe if enabled
	if r.gzip && r.Request.Body != nil {
		r.Request.Header.Set("Content-Encoding", "gzip")
		originalBody := r.Request.Body
		pr, pw := io.Pipe()
		r.Request.Body = pr

		go func() {
			defer pw.Close()
			gw := gzip.NewWriter(pw)
			defer gw.Close()

			_, _ = io.Copy(gw, originalBody)
			_ = originalBody.Close()
		}()
	}

	return r.sendRequest()
}


func (r *Request) sendRequest() (*http.Response, error) {
	resp, err := r.client.Do(r.Request)
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
func (r *Request) Header(k, v string) *Request { r.Request.Header.Set(k, v); return r }

// PathParam replaces a path variable placeholder in the URL.
func (r *Request) PathParam(key, val string) *Request {
	r.URL.Path = strings.ReplaceAll(r.URL.Path, "{"+key+"}", val)
	return r
}
func (r *Request) PathInt(k string, v int) *Request   { return r.PathParam(k, strconv.Itoa(v)) }
func (r *Request) PathBool(k string, v bool) *Request { return r.PathParam(k, strconv.FormatBool(v)) }
func (r *Request) PathFloat(k string, v float64) *Request {
	return r.PathParam(k, strconv.FormatFloat(v, 'f', -1, 64))
}

// Param adds a query parameter to the request.
func (r *Request) Param(key, value string) *Request {
	q := r.URL.Query()
	q.Set(key, value)
	r.URL.RawQuery = q.Encode()
	return r
}
func (r *Request) Int(k string, v int) *Request   { return r.Param(k, strconv.Itoa(v)) }
func (r *Request) Bool(k string, v bool) *Request { return r.Param(k, strconv.FormatBool(v)) }
func (r *Request) Float(k string, v float64) *Request {
	return r.Param(k, strconv.FormatFloat(v, 'f', -1, 64))
}

// Body sets the request body and Content-Type header.
func (r *Request) Body(body io.ReadCloser, contentType string) *Request {
	r.Request.Header.Set("Content-Type", contentType)
	r.Request.Body = body
	return r
}

// JSON sets the request body as JSON.
func (r *Request) JSON(body any) *Request {
	r.Request.Header.Set("Content-Type", string(applicationJSON))
	r.body.contentType = applicationJSON
	r.body.content = body
	return r
}

// Form sets the request body as form data.
func (r *Request) Form(key, value string) *Request {
	r.Request.Header.Set("Content-Type", string(applicationUrlEncodedForm))
	r.body.contentType = applicationUrlEncodedForm
	if r.body.form == nil {
		r.body.form = make(url.Values)
	}
	r.body.form.Set(key, value)
	return r
}

// Cookie adds a cookie to the request.
func (r *Request) Cookie(name, value string) *Request {
	r.AddCookie(&http.Cookie{Name: name, Value: value})
	return r
}
