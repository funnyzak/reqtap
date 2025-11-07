package web

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/funnyzak/reqtap/pkg/request"
)

func TestExportRequestsText(t *testing.T) {
	timestamp := time.Date(2025, time.November, 7, 12, 0, 0, 0, time.UTC)
	items := []*StoredRequest{
		{
			ID: "REQ1",
			RequestData: &request.RequestData{
				Timestamp:  timestamp,
				Method:     "post",
				Path:       "/hook",
				Query:      "a=1",
				RemoteAddr: "127.0.0.1",
				Headers: http.Header{
					"Content-Type": []string{"application/json"},
					"Host":         []string{"example.com"},
				},
				Body: []byte(`{"foo":"bar"}`),
			},
		},
		{
			ID: "REQ2",
			RequestData: &request.RequestData{
				Timestamp: timestamp,
				Method:    "get",
				Path:      "/binary",
				Headers: http.Header{
					"User-Agent": []string{"reqtap"},
				},
				Body:     []byte{0x00, 0x01, 0x02},
				IsBinary: true,
			},
		},
	}

	buf, contentType, ext, err := ExportRequests(items, "txt")
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}

	if contentType != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", contentType)
	}

	if ext != "txt" {
		t.Fatalf("unexpected extension: %s", ext)
	}

	got := string(buf)
	if !strings.Contains(got, "POST /hook?a=1 HTTP/1.1") {
		t.Fatalf("request line missing: %s", got)
	}
	if !strings.Contains(got, "Content-Type: application/json") {
		t.Fatalf("headers missing: %s", got)
	}
	if !strings.Contains(got, "{\"foo\":\"bar\"}") {
		t.Fatalf("body missing: %s", got)
	}
	if !strings.Contains(got, "[binary payload omitted") {
		t.Fatalf("binary placeholder missing: %s", got)
	}
}
