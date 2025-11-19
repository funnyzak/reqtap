package printer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/funnyzak/reqtap/internal/config"
	"github.com/funnyzak/reqtap/pkg/request"
)

func init() {
	color.NoColor = true
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...interface{}) {}
func (noopLogger) Info(string, ...interface{})  {}
func (noopLogger) Warn(string, ...interface{})  {}
func (noopLogger) Error(string, ...interface{}) {}
func (noopLogger) Fatal(string, ...interface{}) {}

func TestConsolePrinter_PrintRequest(t *testing.T) {
	p := NewConsolePrinter(noopLogger{}, nil)
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

func TestConsolePrinter_JSONPretty(t *testing.T) {
	cfg := config.BodyViewConfig{
		Enable:          true,
		MaxPreviewBytes: 0,
		Json: config.JSONViewConfig{
			Enable:         true,
			Pretty:         true,
			MaxIndentBytes: 1024,
		},
	}
	p := NewConsolePrinter(noopLogger{}, &cfg)
	buf := &bytes.Buffer{}
	p.out = buf
	req := &request.RequestData{
		Method:      "POST",
		Path:        "/json",
		Headers:     map[string][]string{"Content-Type": {"application/json"}},
		Body:        []byte(`{"foo":"bar","nested":{"a":1}}`),
		Timestamp:   time.Now(),
		ContentType: "application/json",
	}
	if err := p.PrintRequest(req); err != nil {
		t.Fatalf("print request failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "\n  \"foo\": \"bar\"") {
		t.Fatalf("expected pretty JSON output, got %s", output)
	}
}

func TestConsolePrinter_FormTable(t *testing.T) {
	cfg := config.BodyViewConfig{
		Enable: true,
		Form: config.FormViewConfig{
			Enable: true,
		},
	}
	p := NewConsolePrinter(noopLogger{}, &cfg)
	buf := &bytes.Buffer{}
	p.out = buf
	req := &request.RequestData{
		Method:      "POST",
		Path:        "/form",
		Headers:     map[string][]string{"Content-Type": {"application/x-www-form-urlencoded"}},
		Body:        []byte("foo=bar&foo=baz&bar=baz"),
		Timestamp:   time.Now(),
		ContentType: "application/x-www-form-urlencoded",
	}
	if err := p.PrintRequest(req); err != nil {
		t.Fatalf("print request failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Form data:") || !strings.Contains(output, "foo │ bar, baz") {
		t.Fatalf("expected form table output, got %s", output)
	}
}

func TestConsolePrinter_TruncationNotice(t *testing.T) {
	cfg := config.BodyViewConfig{
		Enable:          true,
		MaxPreviewBytes: 8,
	}
	p := NewConsolePrinter(noopLogger{}, &cfg)
	buf := &bytes.Buffer{}
	p.out = buf
	req := &request.RequestData{
		Method:      "POST",
		Path:        "/truncate",
		Headers:     map[string][]string{"Content-Type": {"text/plain"}},
		Body:        []byte("0123456789abcdef"),
		Timestamp:   time.Now(),
		ContentType: "text/plain",
	}
	if err := p.PrintRequest(req); err != nil {
		t.Fatalf("print request failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "仅展示前") {
		t.Fatalf("expected truncation notice, got %s", output)
	}
	if strings.Contains(output, "abcdef") {
		t.Fatalf("unexpected full body output when preview limit active")
	}
}

func TestConsolePrinter_BinaryPreviewAndSave(t *testing.T) {
	tdir := t.TempDir()
	cfg := config.BodyViewConfig{
		Enable: true,
		Binary: config.BinaryViewConfig{
			HexPreviewEnable: true,
			HexPreviewBytes:  4,
			SaveToFile:       true,
			SaveDirectory:    tdir,
		},
	}
	p := NewConsolePrinter(noopLogger{}, &cfg)
	buf := &bytes.Buffer{}
	p.out = buf
	req := &request.RequestData{
		ID:          "ABCDEF",
		Method:      "POST",
		Path:        "/bin",
		Body:        []byte{0x00, 0x01, 0x02, 0x03, 0x04},
		Timestamp:   time.Now(),
		ContentType: "application/octet-stream",
		IsBinary:    true,
	}
	if err := p.PrintRequest(req); err != nil {
		t.Fatalf("print request failed: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "Hex preview") {
		t.Fatalf("expected hex preview in output")
	}
	if !strings.Contains(output, "Binary saved to") {
		t.Fatalf("expected binary save notice")
	}
	entries, err := os.ReadDir(tdir)
	if err != nil {
		t.Fatalf("read dir failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one saved binary file, got %d", len(entries))
	}
	path := filepath.Join(tdir, entries[0].Name())
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file failed: %v", err)
	}
	if string(content) != string(req.Body) {
		t.Fatalf("saved binary content mismatch")
	}
}
