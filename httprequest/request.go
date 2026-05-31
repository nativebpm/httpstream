package httprequest

import (
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
	client     http.Client
	body       requestPayload
	cancelFunc context.CancelFunc
}

// NewRequest creates a new HTTP request builder.
func NewRequest(ctx context.Context, client http.Client, method string, url string) *Request {
	request, _ := http.NewRequestWithContext(ctx, method, url, nil)
	return &Request{
		Request: request,
		client:  client,
	}
}

// Timeout sets a timeout for the request.
func (r *Request) Timeout(duration time.Duration) *Request {
	ctx, cancel := context.WithTimeout(r.Context(), duration)
	r.cancelFunc = cancel
	r.Request = r.WithContext(ctx)
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
func (r *Request) Header(key, value string) *Request {
	r.Request.Header.Set(key, value)
	return r
}

// PathParam replaces a path variable placeholder in the URL.
// Replaces {key} with the provided value.
// Example: "/users/{id}" with PathParam("id", "123") becomes "/users/123"
func (r *Request) PathParam(key, value string) *Request {
	placeholder := "{" + key + "}"
	r.Request.URL.Path = strings.ReplaceAll(r.Request.URL.Path, placeholder, value)
	return r
}

// PathInt replaces a path variable placeholder with an integer value.
func (r *Request) PathInt(key string, value int) *Request {
	return r.PathParam(key, strconv.Itoa(value))
}

// PathBool replaces a path variable placeholder with a boolean value.
func (r *Request) PathBool(key string, value bool) *Request {
	return r.PathParam(key, strconv.FormatBool(value))
}

// PathFloat replaces a path variable placeholder with a float64 value.
func (r *Request) PathFloat(key string, value float64) *Request {
	return r.PathParam(key, strconv.FormatFloat(value, 'f', -1, 64))
}

// Param adds a query parameter to the request.
func (r *Request) Param(key, value string) *Request {
	q := r.Request.URL.Query()
	q.Set(key, value)
	r.Request.URL.RawQuery = q.Encode()
	return r
}

// Bool adds a boolean query parameter to the request.
func (r *Request) Bool(key string, value bool) *Request {
	return r.Param(key, strconv.FormatBool(value))
}

// Float adds a float64 query parameter to the request.
func (r *Request) Float(key string, value float64) *Request {
	return r.Param(key, strconv.FormatFloat(value, 'f', -1, 64))
}

// Int adds an integer query parameter to the request.
func (r *Request) Int(key string, value int) *Request {
	return r.Param(key, strconv.Itoa(value))
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
// Subsequent calls append additional cookies.
func (r *Request) Cookie(name, value string) *Request {
	r.Request.AddCookie(&http.Cookie{Name: name, Value: value})
	return r
}
