package web

import (
	"strconv"
	"strings"
	"sync"

	"github.com/funnyzak/reqtap/pkg/request"
)

// StoredRequest wraps RequestData with a unique identifier.
type StoredRequest struct {
	ID string `json:"id"`
	*request.RequestData
}

// ListOptions describes filters for querying stored requests.
type ListOptions struct {
	Search string
	Method string
	Limit  int
	Offset int
}

// RequestStore keeps recent requests in-memory using a ring buffer.
type RequestStore struct {
	mu      sync.RWMutex
	max     int
	counter uint64
	items   []*StoredRequest
}

// NewRequestStore creates a new RequestStore with the provided capacity.
func NewRequestStore(max int) *RequestStore {
	if max < 1 {
		max = 1
	}

	return &RequestStore{
		max:   max,
		items: make([]*StoredRequest, 0, max),
	}
}

// Add stores a new request and returns the stored record.
func (s *RequestStore) Add(data *request.RequestData) *StoredRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counter++
	record := &StoredRequest{
		ID:          generateRequestID(s.counter),
		RequestData: data,
	}

	if len(s.items) >= s.max {
		// Drop oldest
		s.items = append(s.items[1:], record)
	} else {
		s.items = append(s.items, record)
	}

	return record
}

// List returns filtered requests (newest first) along with the total count.
func (s *RequestStore) List(opts ListOptions) ([]*StoredRequest, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	search := strings.ToLower(strings.TrimSpace(opts.Search))
	method := strings.ToUpper(strings.TrimSpace(opts.Method))

	filtered := make([]*StoredRequest, 0, len(s.items))
	for i := len(s.items) - 1; i >= 0; i-- {
		item := s.items[i]

		if method != "" && strings.ToUpper(item.Method) != method {
			continue
		}

		if search != "" && !matchesSearch(item, search) {
			continue
		}

		filtered = append(filtered, item)
	}

	total := len(filtered)

	limit := opts.Limit
	if limit <= 0 || limit > total {
		limit = total
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return filtered[offset:end], total
}

// Snapshot returns all stored requests (newest first).
func (s *RequestStore) Snapshot() []*StoredRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*StoredRequest, 0, len(s.items))
	for i := len(s.items) - 1; i >= 0; i-- {
		result = append(result, s.items[i])
	}
	return result
}

// Get locates a request by id.
func (s *RequestStore) Get(id string) (*StoredRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := len(s.items) - 1; i >= 0; i-- {
		if s.items[i].ID == id {
			return s.items[i], true
		}
	}
	return nil, false
}

func matchesSearch(item *StoredRequest, term string) bool {
	target := strings.ToLower(
		item.Path + " " +
			item.Query + " " +
			item.RemoteAddr + " " +
			item.UserAgent,
	)

	if strings.Contains(target, term) {
		return true
	}

	// Search headers keys/values
	for key, values := range item.Headers {
		if strings.Contains(strings.ToLower(key), term) {
			return true
		}
		for _, val := range values {
			if strings.Contains(strings.ToLower(val), term) {
				return true
			}
		}
	}

	return false
}

func generateRequestID(counter uint64) string {
	return strings.ToUpper(strconv.FormatUint(counter, 36))
}
