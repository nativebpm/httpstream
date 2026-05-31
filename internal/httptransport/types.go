package httptransport

import (
	"net/http"
)

type Middleware func(http.RoundTripper) http.RoundTripper
