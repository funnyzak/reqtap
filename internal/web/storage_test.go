package web

import (
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/funnyzak/reqtap/pkg/request"
)

func TestRequestStore_AddAndGet(t *testing.T) {
	store := NewRequestStore(2)
	req1 := fakeRequest("GET", "/a", "1")
	req2 := fakeRequest("POST", "/b", "2")
	req3 := fakeRequest("GET", "/c", "3")

	item1 := store.Add(req1)
	item2 := store.Add(req2)
	// overwrite oldest
	store.Add(req3)

	if _, ok := store.Get(item1.ID); ok {
		t.Fatalf("expected oldest request to be evicted")
	}

	if got, ok := store.Get(item2.ID); !ok || got.ID != item2.ID {
		t.Fatalf("expected to retrieve item2")
	}
}

func TestRequestStore_ListFiltersAndPaging(t *testing.T) {
	store := NewRequestStore(5)
	methods := []string{"GET", "POST", "GET", "PUT", "GET"}
	for i, m := range methods {
		store.Add(fakeRequest(m, "/p"+strconv.Itoa(i), strconv.Itoa(i)))
	}

	items, total := store.List(ListOptions{Method: "GET", Limit: 2, Offset: 0})
	if total != 3 {
		t.Fatalf("expected 3 GET items, got %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items page size, got %d", len(items))
	}

	// search should be case-insensitive
	items, total = store.List(ListOptions{Search: "P3"})
	if total != 1 || !strings.Contains(items[0].Path, "3") {
		t.Fatalf("search filter failed")
	}
}

func TestRequestStore_SnapshotOrder(t *testing.T) {
	store := NewRequestStore(3)
	store.Add(fakeRequest("GET", "/a", "1"))
	store.Add(fakeRequest("GET", "/b", "2"))
	store.Add(fakeRequest("GET", "/c", "3"))

	snap := store.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("expected snapshot size 3, got %d", len(snap))
	}
	if snap[0].Path != "/c" || snap[2].Path != "/a" {
		t.Fatalf("snapshot order should be newest first")
	}
}

func TestRequestStore_IterateStops(t *testing.T) {
	store := NewRequestStore(2)
	store.Add(fakeRequest("GET", "/a", "1"))
	store.Add(fakeRequest("GET", "/b", "2"))

	count := 0
	store.Iterate(ListOptions{}, func(*StoredRequest) bool {
		count++
		return false
	})
	if count != 1 {
		t.Fatalf("iterate should stop on false")
	}
}

func TestRequestStore_ConcurrentReads(t *testing.T) {
	store := NewRequestStore(10)
	for i := 0; i < 10; i++ {
		store.Add(fakeRequest("GET", "/p"+strconv.Itoa(i), strconv.Itoa(i)))
	}

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			store.List(ListOptions{Method: "GET"})
		}()
	}
	wg.Wait()
}

func fakeRequest(method, path, body string) *request.RequestData {
	return &request.RequestData{
		Method:        method,
		Path:          path,
		Body:          []byte(body),
		ContentLength: int64(len(body)),
	}
}
