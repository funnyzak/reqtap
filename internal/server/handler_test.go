package server

import (
	"net/http/httptest"
	"testing"
)

func TestSelectResponseRule(t *testing.T) {
	h := &Handler{
		config: &ServerConfig{
			Responses: []ImmediateResponseRule{
				{Name: "exact", Path: "/foo", Status: 201},
				{Name: "prefix", PathPrefix: "/bar", Status: 202},
				{Name: "method", Methods: []string{"POST"}, Status: 203},
			},
		},
	}

	req := httptest.NewRequest("GET", "http://localhost/foo", nil)
	rule := h.selectResponseRule(req)
	if rule == nil || rule.Name != "exact" {
		t.Fatalf("expected exact rule, got %#v", rule)
	}

	req = httptest.NewRequest("GET", "http://localhost/bar/baz", nil)
	rule = h.selectResponseRule(req)
	if rule == nil || rule.Name != "prefix" {
		t.Fatalf("expected prefix rule, got %#v", rule)
	}

	req = httptest.NewRequest("POST", "http://localhost/any", nil)
	rule = h.selectResponseRule(req)
	if rule == nil || rule.Name != "method" {
		t.Fatalf("expected method rule, got %#v", rule)
	}
}

func TestSendImmediateResponse(t *testing.T) {
	h := &Handler{
		logger: noopLogger{},
		config: &ServerConfig{
			Responses: []ImmediateResponseRule{
				{
					Name:   "json",
					Path:   "/json",
					Status: 202,
					Body:   "{\"ok\":true}",
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				},
			},
		},
	}

	req := httptest.NewRequest("GET", "http://localhost/json", nil)
	rr := httptest.NewRecorder()
	h.sendImmediateResponse(rr, req)

	if rr.Code != 202 {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type json, got %s", ct)
	}
	if body := rr.Body.String(); body != "{\"ok\":true}" {
		t.Fatalf("unexpected body %s", body)
	}
}

// noopLogger implements logger.Logger for tests
type noopLogger struct{}

func (noopLogger) Debug(string, ...interface{}) {}
func (noopLogger) Info(string, ...interface{})  {}
func (noopLogger) Warn(string, ...interface{})  {}
func (noopLogger) Error(string, ...interface{}) {}
func (noopLogger) Fatal(string, ...interface{}) {}
