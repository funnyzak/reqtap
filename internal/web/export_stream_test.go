package web

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/funnyzak/reqtap/pkg/request"
)

func TestStreamExportJSON(t *testing.T) {
	items := []*StoredRequest{{
		ID:          "1",
		RequestData: &RequestDataFixture,
	}}
	buf := &bytes.Buffer{}
	iter := func(yield func(*StoredRequest) bool) {
		for _, it := range items {
			yield(it)
		}
	}
	ct, ext, err := StreamExport(buf, iter, "json")
	if err != nil {
		t.Fatalf("stream export failed: %v", err)
	}
	if ct != "application/json" || ext != "json" {
		t.Fatalf("unexpected metadata: %s %s", ct, ext)
	}
	if !strings.HasPrefix(buf.String(), "[") {
		t.Fatalf("json should start with [")
	}
}

var RequestDataFixture = request.RequestData{
	Method:        "POST",
	Path:          "/hook",
	Timestamp:     time.Unix(0, 0),
	Headers:       map[string][]string{"User-Agent": {"ua"}},
	Body:          []byte("demo"),
	ContentType:   "text/plain",
	ContentLength: 4,
}

func TestStreamExportCSV(t *testing.T) {
	items := []*StoredRequest{{ID: "1", RequestData: &RequestDataFixture}}
	buf := &bytes.Buffer{}
	iter := func(yield func(*StoredRequest) bool) {
		for _, it := range items {
			yield(it)
		}
	}
	_, _, err := StreamExport(buf, iter, "csv")
	if err != nil {
		t.Fatalf("csv export failed: %v", err)
	}
	if !strings.Contains(buf.String(), "id,timestamp") {
		t.Fatalf("csv header missing")
	}
}

func TestStreamExportText(t *testing.T) {
	items := []*StoredRequest{{ID: "1", RequestData: &RequestDataFixture}}
	buf := &bytes.Buffer{}
	iter := func(yield func(*StoredRequest) bool) {
		for _, it := range items {
			yield(it)
		}
	}
	_, _, err := StreamExport(buf, iter, "txt")
	if err != nil {
		t.Fatalf("txt export failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Request 1") {
		t.Fatalf("text export missing content")
	}
}

func TestDescribeFormatInvalid(t *testing.T) {
	if _, _, err := StreamExport(&bytes.Buffer{}, func(func(*StoredRequest) bool) {}, "xml"); err == nil {
		t.Fatalf("expected error for unsupported format")
	}
}
