package httpstream

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestConcurrencyLimit(t *testing.T) {
	// Semaphore limit is 2
	middleware := ConcurrencyMiddleware(2)

	var wg sync.WaitGroup
	var activeCount int
	var maxActive int
	var mu sync.Mutex

	// Mock roundtripper that simulates work by sleeping
	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			mu.Lock()
			activeCount++
			if activeCount > maxActive {
				maxActive = activeCount
			}
			mu.Unlock()

			// Simulate request duration
			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			activeCount--
			mu.Unlock()

			return &http.Response{StatusCode: 200}, nil
		},
	}

	limiter := middleware(mockRT)

	// Launch 5 concurrent requests
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
			resp, err := limiter.RoundTrip(req)
			assert.NoError(t, err)
			assert.Equal(t, 200, resp.StatusCode)
		}()
	}

	wg.Wait()

	// Max concurrent active requests must not exceed 2
	assert.True(t, maxActive <= 2, "max active requests was %d, expected <= 2", maxActive)
}

func TestConcurrencyLimitContextCancellation(t *testing.T) {
	// Semaphore limit is 1
	middleware := ConcurrencyMiddleware(1)

	// Block channel to keep the first request active
	blockChan := make(chan struct{})

	mockRT := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			<-blockChan
			return &http.Response{StatusCode: 200}, nil
		},
	}

	limiter := middleware(mockRT)

	// Start 1st request (occupies the only slot)
	go func() {
		req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://example.com", nil)
		_, _ = limiter.RoundTrip(req)
	}()

	// Give the 1st request time to start and acquire the slot
	time.Sleep(10 * time.Millisecond)

	// Start 2nd request with a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	req2, _ := http.NewRequestWithContext(ctx, "GET", "http://example.com", nil)

	var err2 error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err2 = limiter.RoundTrip(req2)
	}()

	// Give the 2nd request time to block on acquiring slot
	time.Sleep(10 * time.Millisecond)

	// Cancel the context of the 2nd request
	cancel()

	wg.Wait()

	// Verify that the 2nd request failed with context.Canceled immediately
	assert.Error(t, err2)
	assert.True(t, errors.Is(err2, context.Canceled), "expected context.Canceled, got %v", err2)

	// Cleanup block channel to release the 1st request
	close(blockChan)
}
