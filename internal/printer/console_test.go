package printer

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/funnyzak/reqtap/pkg/request"
)

type noopLogger struct{}

func (noopLogger) Debug(string, ...interface{}) {}
func (noopLogger) Info(string, ...interface{})  {}
func (noopLogger) Warn(string, ...interface{})  {}
func (noopLogger) Error(string, ...interface{}) {}
func (noopLogger) Fatal(string, ...interface{}) {}

func TestConsolePrinter_PrintRequest(t *testing.T) {
	p := NewConsolePrinter(noopLogger{})
	buf := &bytes.Buffer{}
	p.out = buf
	os.Setenv("REQTAP_TEST_WIDTH", "80")

	req := &request.RequestData{
		Method:        "GET",
		Path:          "/hello",
		Query:         "q=1",
		Proto:         "HTTP/1.1",
		Headers:       map[string][]string{"User-Agent": {"test"}, "Authorization": {"secret"}},
		Body:          []byte("hi"),
		Timestamp:     time.Now(),
		ContentType:   "text/plain",
		ContentLength: 2,
	}

	if err := p.PrintRequest(req); err != nil {
		t.Fatalf("print request failed: %v", err)
	}

	if !bytes.Contains(buf.Bytes(), []byte("Request #")) {
		t.Fatalf("output missing summary")
	}
	if bytes.Contains(buf.Bytes(), []byte("secret")) {
		t.Fatalf("sensitive header should be redacted")
	}
	if !bytes.Contains(buf.Bytes(), []byte("GET")) {
		t.Fatalf("method line missing")
	}
}
