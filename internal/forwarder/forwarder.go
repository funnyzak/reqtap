package forwarder

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/pkg/request"
)

// Forwarder request forwarder
type Forwarder struct {
	client        *http.Client
	logger        logger.Logger
	timeout       time.Duration
	retries       int
	maxConcurrent int
	workerPool    chan struct{}
	mu            sync.Mutex
	cond          *sync.Cond
	closed        bool
	activeCalls   int
}

// ErrForwarderClosed indicates the forwarder has been shut down.
var ErrForwarderClosed = errors.New("forwarder is closed")

// NewForwarder creates new forwarder
func NewForwarder(logger logger.Logger, timeout time.Duration, retries, maxConcurrent int) *Forwarder {
	if maxConcurrent <= 0 {
		maxConcurrent = 10 // Default max concurrent count
	}

	f := &Forwarder{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger:        logger,
		timeout:       timeout,
		retries:       retries,
		maxConcurrent: maxConcurrent,
		workerPool:    make(chan struct{}, maxConcurrent),
	}
	f.cond = sync.NewCond(&f.mu)
	return f
}

// Forward forwards request to all configured URLs
func (f *Forwarder) Forward(ctx context.Context, data *request.RequestData, urls []string) error {
	if len(urls) == 0 {
		return nil
	}

	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return ErrForwarderClosed
	}
	f.activeCalls++
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.activeCalls--
		if f.activeCalls == 0 {
			f.cond.Broadcast()
		}
		f.mu.Unlock()
	}()

	// Concurrently forward to all target URLs
	var wg sync.WaitGroup
	for _, url := range urls {
		wg.Add(1)
		go func(targetURL string) {
			defer wg.Done()

			// Get worker token (control concurrent count)
			f.workerPool <- struct{}{}
			defer func() { <-f.workerPool }()

			f.forwardToURL(ctx, data, targetURL)
		}(url)
	}

	wg.Wait()
	return nil
}

// forwardToURL forwards request to single URL (with retry)
func (f *Forwarder) forwardToURL(ctx context.Context, data *request.RequestData, targetURL string) {
	var lastErr error

	for attempt := 0; attempt <= f.retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second // Maximum backoff time
			}

			select {
			case <-ctx.Done():
				f.logger.Info("Forward cancelled by context",
					"url", targetURL,
					"attempt", attempt+1,
				)
				return
			case <-time.After(backoff):
				// Continue retry
			}
		}

		err := f.doForward(ctx, data, targetURL, attempt)
		if err == nil {
			f.logger.Info("Request forwarded successfully",
				"url", targetURL,
				"method", data.Method,
				"path", data.Path,
				"attempt", attempt+1,
			)
			return
		}

		lastErr = err
		f.logger.Warn("Forward attempt failed",
			"url", targetURL,
			"error", err.Error(),
			"attempt", attempt+1,
		)
	}

	f.logger.Error("All forward attempts failed",
		"url", targetURL,
		"final_error", lastErr.Error(),
		"total_attempts", f.retries+1,
	)
}

// doForward executes single forward
func (f *Forwarder) doForward(ctx context.Context, data *request.RequestData, targetURL string, attempt int) error {
	// Build target URL
	targetURL = strings.TrimSuffix(targetURL, "/") + data.Path
	if data.Query != "" {
		targetURL += "?" + data.Query
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, data.Method, targetURL, bytes.NewReader(data.Body))
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	// Copy Headers (filter some headers that should not be forwarded)
	for key, values := range data.Headers {
		if f.shouldForwardHeader(key) {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
	}

	// Set X-Forwarded-* headers
	req.Header.Set("X-Forwarded-For", data.RemoteAddr)
	req.Header.Set("X-Forwarded-Proto", "http")
	req.Header.Set("X-ReqTap-Original-Host", data.Headers.Get("Host"))
	req.Header.Set("X-ReqTap-Forward-Attempt", fmt.Sprintf("%d", attempt+1))

	// Send request
	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response (avoid connection pool issues)
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		f.logger.Warn("Failed to read response body", "error", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("target returned status %d", resp.StatusCode)
	}

	return nil
}

// shouldForwardHeader determines if specified header should be forwarded
func (f *Forwarder) shouldForwardHeader(key string) bool {
	lowerKey := strings.ToLower(key)

	// Headers that should not be forwarded
	skipHeaders := map[string]bool{
		"host":                true, // Automatically set
		"connection":          true, // Connection related
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailers":            true,
		"transfer-encoding":   true,
		"upgrade":             true,
		"content-length":      true, // Will be automatically recalculated
	}

	if skipHeaders[lowerKey] {
		return false
	}

	// For some sensitive headers, log warning but still forward (can be adjusted as needed)
	sensitiveHeaders := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"set-cookie":    true,
	}

	if sensitiveHeaders[lowerKey] {
		f.logger.Debug("Forwarding sensitive header", "header", key)
	}

	return true
}

// SetMaxConcurrent sets maximum concurrent count
func (f *Forwarder) SetMaxConcurrent(maxConcurrent int) {
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	f.maxConcurrent = maxConcurrent
	newWorkerPool := make(chan struct{}, maxConcurrent)
	close(f.workerPool)
	f.workerPool = newWorkerPool
}

// GetMaxConcurrent gets current maximum concurrent count
func (f *Forwarder) GetMaxConcurrent() int {
	return f.maxConcurrent
}

// Close closes forwarder and cleans up resources
func (f *Forwarder) Close() {
	f.mu.Lock()
	if f.closed {
		f.mu.Unlock()
		return
	}
	f.closed = true
	for f.activeCalls > 0 {
		f.cond.Wait()
	}
	f.mu.Unlock()

	close(f.workerPool)

	// Close idle connections of HTTP client
	if transport, ok := f.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
}
