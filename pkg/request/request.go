package request

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RequestData represents received HTTP request data
type RequestData struct {
	ID            string       `json:"id"`
	Timestamp     time.Time    `json:"timestamp"`
	Method        string       `json:"method"`
	Proto         string       `json:"proto"`
	Path          string       `json:"path"`
	Query         string       `json:"query"`
	RemoteAddr    string       `json:"remote_addr"`
	UserAgent     string       `json:"user_agent"`
	Headers       http.Header  `json:"headers"`
	Body          []byte       `json:"body"`
	ContentType   string       `json:"content_type"`
	ContentLength int64        `json:"content_length"`
	IsBinary      bool         `json:"is_binary"`
	Size          int64        `json:"size"`
	MockResponse  MockResponse `json:"mock_response"`
}

// MockResponse summarizes inline response meta
type MockResponse struct {
	Rule   string `json:"rule"`
	Status int    `json:"status"`
}

// NewRequestData creates new request data record
func NewRequestData(r *http.Request, body []byte) *RequestData {
	id := generateRequestID()
	contentType := r.Header.Get("Content-Type")

	return &RequestData{
		ID:            id,
		Timestamp:     time.Now(),
		Method:        r.Method,
		Proto:         r.Proto,
		Path:          r.URL.Path,
		Query:         r.URL.RawQuery,
		RemoteAddr:    getClientIP(r),
		UserAgent:     r.UserAgent(),
		Headers:       r.Header.Clone(),
		Body:          body,
		ContentType:   contentType,
		ContentLength: r.ContentLength,
		IsBinary:      isBinaryContent(contentType, body),
		Size:          int64(len(body)),
	}
}

// getClientIP gets client real IP address
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client IP)
		for idx := 0; idx < len(xff); idx++ {
			if xff[idx] == ',' {
				return xff[:idx]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Use RemoteAddr
	if idx := len(r.RemoteAddr) - 1; idx >= 0 && r.RemoteAddr[idx] >= '0' && r.RemoteAddr[idx] <= '9' {
		// Find colon from back to front
		for i := idx; i >= 0; i-- {
			if r.RemoteAddr[i] == ':' {
				return r.RemoteAddr[:i]
			}
		}
	}

	return r.RemoteAddr
}

// isBinaryContent detects if it's binary content
func isBinaryContent(contentType string, body []byte) bool {
	// Check Content-Type
	binaryTypes := []string{
		"image/", "video/", "audio/",
		"application/octet-stream",
		"application/zip", "application/gzip",
		"application/pdf", "application/msword",
		"application/vnd.ms-", "application/vnd.openxmlformats-",
	}

	for _, binaryType := range binaryTypes {
		if strings.HasPrefix(contentType, binaryType) {
			return true
		}
	}

	// Check null byte ratio
	nullCount := 0
	for _, b := range body {
		if b == 0 {
			nullCount++
		}
	}
	if len(body) > 0 && nullCount > len(body)/10 { // More than 10% are null bytes
		return true
	}

	return false
}

// generateRequestID creates a random, URL-safe request identifier.
func generateRequestID() string {
	const idBytes = 12 // 12 bytes => 24 hex characters, compact but unique enough
	b := make([]byte, idBytes)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based value to avoid returning empty ID
		return fmt.Sprintf("REQ-%d", time.Now().UnixNano())
	}
	return strings.ToUpper(hex.EncodeToString(b))
}
