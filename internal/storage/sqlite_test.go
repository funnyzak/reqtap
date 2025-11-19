package storage

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/funnyzak/reqtap/internal/config"
	"github.com/funnyzak/reqtap/pkg/request"
)

type noopLogger struct{}

func (noopLogger) Debug(string, ...interface{}) {}
func (noopLogger) Info(string, ...interface{})  {}
func (noopLogger) Warn(string, ...interface{})  {}
func (noopLogger) Error(string, ...interface{}) {}
func (noopLogger) Fatal(string, ...interface{}) {}

func newTestStore(t *testing.T, maxRecords int) Store {
	t.Helper()
	dir := t.TempDir()
	cfg := &config.StorageConfig{
		Driver:     "sqlite",
		Path:       filepath.Join(dir, "reqtap.db"),
		MaxRecords: maxRecords,
	}
	store, err := New(cfg, noopLogger{})
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() {
		store.Close()
	})
	return store
}

func fakeRequest(id, method, path string) *request.RequestData {
	return &request.RequestData{
		ID:        id,
		Timestamp: time.Now(),
		Method:    method,
		Path:      path,
		Headers:   http.Header{"User-Agent": []string{"reqtap"}},
		Body:      []byte("body"),
	}
}

func TestSQLiteStore_RecordAndGet(t *testing.T) {
	store := newTestStore(t, 100)
	data := fakeRequest("rec-1", "POST", "/hook")
	rec, err := store.Record(data)
	if err != nil {
		t.Fatalf("record failed: %v", err)
	}
	if rec.ID == "" {
		t.Fatal("expected record id to be set")
	}
	got, err := store.Get(rec.ID)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got == nil || got.Method != "POST" {
		t.Fatalf("unexpected record returned: %#v", got)
	}
	if string(got.Body) != "body" {
		t.Fatalf("unexpected body: %s", string(got.Body))
	}
}

func TestSQLiteStore_ListFilters(t *testing.T) {
	store := newTestStore(t, 100)
	methods := []string{"GET", "POST", "GET"}
	for i, method := range methods {
		req := fakeRequest(fmt.Sprintf("rec-%d", i), method, fmt.Sprintf("/p%d", i))
		req.Query = "q=demo"
		if _, err := store.Record(req); err != nil {
			t.Fatalf("record failed: %v", err)
		}
	}

	items, total, err := store.List(ListOptions{Method: "POST"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("expected 1 POST record, got total=%d len=%d", total, len(items))
	}

	items, total, err = store.List(ListOptions{Search: "p"})
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 results, got %d", total)
	}
}

func TestSQLiteStore_IterateStops(t *testing.T) {
	store := newTestStore(t, 100)
	for i := 0; i < 5; i++ {
		if _, err := store.Record(fakeRequest(fmt.Sprintf("rec-%d", i), "GET", fmt.Sprintf("/i%d", i))); err != nil {
			t.Fatalf("record failed: %v", err)
		}
	}
	count := 0
	err := store.Iterate(ListOptions{}, func(*StoredRequest) bool {
		count++
		return count < 3
	})
	if err != nil {
		t.Fatalf("iterate failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected to stop after 3 iterations, got %d", count)
	}
}

func TestSQLiteStore_PruneMaxRecords(t *testing.T) {
	store := newTestStore(t, 2)
	for i := 0; i < 3; i++ {
		if _, err := store.Record(fakeRequest(fmt.Sprintf("rec-%d", i), "GET", fmt.Sprintf("/p%d", i))); err != nil {
			t.Fatalf("record failed: %v", err)
		}
	}
	items, total, err := store.List(ListOptions{})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("expected only 2 records retained, got total=%d len=%d", total, len(items))
	}
}
