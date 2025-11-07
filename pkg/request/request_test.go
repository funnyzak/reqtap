package request

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewRequestData(t *testing.T) {
	// Create test request
	body := strings.NewReader(`{"test": "data"}`)
	req, err := http.NewRequest("POST", "/test/path?param=value", body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("X-Forwarded-For", "192.168.1.100")
	req.RemoteAddr = "10.0.0.1:12345"

	// Read request body
	reqBody, err := readRequestBody(req)
	if err != nil {
		t.Fatalf("Failed to read request body: %v", err)
	}

	// Create request data
	data := NewRequestData(req, reqBody)

	// Verify basic fields
	if data.Method != "POST" {
		t.Errorf("Expected method POST, got %s", data.Method)
	}

	if data.Proto != "HTTP/1.1" {
		t.Errorf("Expected proto HTTP/1.1, got %s", data.Proto)
	}

	if data.Path != "/test/path" {
		t.Errorf("Expected path /test/path, got %s", data.Path)
	}

	if data.Query != "param=value" {
		t.Errorf("Expected query param=value, got %s", data.Query)
	}

	if data.UserAgent != "test-agent" {
		t.Errorf("Expected User-Agent test-agent, got %s", data.UserAgent)
	}

	if data.ContentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", data.ContentType)
	}

	if string(data.Body) != `{"test": "data"}` {
		t.Errorf("Expected body {\"test\": \"data\"}, got %s", string(data.Body))
	}

	if data.Size != 16 {
		t.Errorf("Expected size 16, got %d", data.Size)
	}

	// Verify IP getting logic (should prioritize X-Forwarded-For)
	if data.RemoteAddr != "192.168.1.100" {
		t.Errorf("Expected remote addr 192.168.1.100, got %s", data.RemoteAddr)
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For single IP",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100",
			},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.100, 10.0.0.2, 172.16.0.1",
			},
			expectedIP: "192.168.1.100",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			headers: map[string]string{
				"X-Real-IP": "192.168.1.200",
			},
			expectedIP: "192.168.1.200",
		},
		{
			name:       "RemoteAddr only",
			remoteAddr: "10.0.0.1:12345",
			headers:    map[string]string{},
			expectedIP: "10.0.0.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "10.0.0.1",
			headers:    map[string]string{},
			expectedIP: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				RemoteAddr: tt.remoteAddr,
				Header:     make(http.Header),
			}

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("Expected IP %s, got %s", tt.expectedIP, ip)
			}
		})
	}
}

func TestIsBinaryContent(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		expected    bool
	}{
		{
			name:        "JSON content",
			contentType: "application/json",
			body:        []byte(`{"key": "value"}`),
			expected:    false,
		},
		{
			name:        "JPEG image",
			contentType: "image/jpeg",
			body:        []byte{0xFF, 0xD8, 0xFF, 0xE0},
			expected:    true,
		},
		{
			name:        "PDF file",
			contentType: "application/pdf",
			body:        []byte("%PDF-1.4"),
			expected:    true,
		},
		{
			name:        "Plain text",
			contentType: "text/plain",
			body:        []byte("Hello, World!"),
			expected:    false,
		},
		{
			name:        "Empty content type with null bytes",
			contentType: "",
			body:        []byte{0x00, 0x00, 0x48, 0x65, 0x6C, 0x6C, 0x6F},
			expected:    true, // More than 10% null bytes
		},
		{
			name:        "Form data",
			contentType: "application/x-www-form-urlencoded",
			body:        []byte("key=value&foo=bar"),
			expected:    false,
		},
		{
			name:        "ZIP file",
			contentType: "application/zip",
			body:        []byte{0x50, 0x4B, 0x03, 0x04},
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isBinaryContent(tt.contentType, tt.body)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for content type %s", tt.expected, result, tt.contentType)
			}
		})
	}
}

// Helper function: read request body
func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return []byte{}, nil
	}
	defer req.Body.Close()

	body := make([]byte, req.ContentLength)
	_, err := req.Body.Read(body)
	return body, err
}

func TestRequestDataTimestamp(t *testing.T) {
	before := time.Now()

	body := strings.NewReader("test")
	req, _ := http.NewRequest("GET", "/", body)
	reqBody, _ := readRequestBody(req)

	data := NewRequestData(req, reqBody)

	after := time.Now()

	if data.Timestamp.Before(before) || data.Timestamp.After(after) {
		t.Errorf("Timestamp %v should be between %v and %v", data.Timestamp, before, after)
	}
}

func TestRequestDataHeaders(t *testing.T) {
	body := strings.NewReader("test")
	req, _ := http.NewRequest("GET", "/", body)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-browser")

	reqBody, _ := readRequestBody(req)
	data := NewRequestData(req, reqBody)

	// Verify headers are properly copied
	if data.Headers.Get("Authorization") != "Bearer token123" {
		t.Error("Authorization header not properly copied")
	}

	if data.Headers.Get("Content-Type") != "application/json" {
		t.Error("Content-Type header not properly copied")
	}

	// Verify original headers are not affected
	if req.Header.Get("User-Agent") != "test-browser" {
		t.Error("Original headers modified")
	}
}

func BenchmarkNewRequestData(b *testing.B) {
	body := strings.NewReader(`{"test": "data", "number": 123}`)
	req, _ := http.NewRequest("POST", "/api/test?param=value", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "benchmark-test")
	req.RemoteAddr = "127.0.0.1:8080"

	reqBody, _ := readRequestBody(req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewRequestData(req, reqBody)
	}
}
