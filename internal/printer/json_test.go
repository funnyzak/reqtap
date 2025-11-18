package printer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/funnyzak/reqtap/pkg/request"
)

func TestJSONPrinter_PrintRequest(t *testing.T) {
	p := NewJSONPrinter(noopLogger{})
	buf := &bytes.Buffer{}
	p.SetOutput(buf)

	data := &request.RequestData{
		Method:    "POST",
		Path:      "/demo",
		Timestamp: time.Now(),
		Headers:   http.Header{"Content-Type": {"application/json"}},
		Body:      []byte("{}"),
	}

	if err := p.PrintRequest(data); err != nil {
		t.Fatalf("print request failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if decoded["type"] != "request" {
		t.Fatalf("unexpected type: %v", decoded["type"])
	}
}
