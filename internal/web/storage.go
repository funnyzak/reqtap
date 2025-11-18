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
	mu          sync.RWMutex
	max         int
	counter     uint64
	items       []*StoredRequest
	head        int
	count       int
	index       map[string]int
	methodIndex map[string][]int
}

// NewRequestStore creates a new RequestStore with the provided capacity.
func NewRequestStore(max int) *RequestStore {
	if max < 1 {
		max = 1
	}

	return &RequestStore{
		max:         max,
		items:       make([]*StoredRequest, max),
		index:       make(map[string]int, max),
		methodIndex: make(map[string][]int),
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

	// Overwrite oldest when buffer满
	if s.count == s.max {
		old := s.items[s.head]
		if old != nil {
			delete(s.index, old.ID)
		}
	}

	s.items[s.head] = record
	s.index[record.ID] = s.head
	s.addMethodIndex(record.Method, s.head)

	s.head = (s.head + 1) % s.max
	if s.count < s.max {
		s.count++
	}

	return record
}

// List returns filtered requests (newest first) along with the total count.
func (s *RequestStore) List(opts ListOptions) ([]*StoredRequest, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	search := strings.ToLower(strings.TrimSpace(opts.Search))
	method := strings.ToUpper(strings.TrimSpace(opts.Method))

	iterPositions := s.collectPositions(method)
	filtered := make([]*StoredRequest, 0, len(iterPositions))
	for _, pos := range iterPositions {
		item := s.items[pos]
		if item == nil || (method != "" && strings.ToUpper(item.Method) != method) {
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

	positions := s.collectPositions("")
	result := make([]*StoredRequest, 0, len(positions))
	for _, pos := range positions {
		if item := s.items[pos]; item != nil {
			result = append(result, item)
		}
	}
	return result
}

// Get locates a request by id.
func (s *RequestStore) Get(id string) (*StoredRequest, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pos, ok := s.index[id]
	if !ok {
		return nil, false
	}
	item := s.items[pos]
	if item == nil || item.ID != id {
		return nil, false
	}
	return item, true
}

// Iterate 按时间倒序遍历，遇到 fn 返回 false 终止
func (s *RequestStore) Iterate(opts ListOptions, fn func(*StoredRequest) bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	search := strings.ToLower(strings.TrimSpace(opts.Search))
	method := strings.ToUpper(strings.TrimSpace(opts.Method))

	positions := s.collectPositions(method)
	for _, pos := range positions {
		item := s.items[pos]
		if item == nil || (method != "" && strings.ToUpper(item.Method) != method) {
			continue
		}
		if search != "" && !matchesSearch(item, search) {
			continue
		}
		if !fn(item) {
			return
		}
	}
}

func (s *RequestStore) collectPositions(method string) []int {
	if s.count == 0 {
		return nil
	}

	// 如果指定方法且有索引，优先按索引定位
	if method != "" {
		positions := s.methodIndex[strings.ToUpper(method)]
		result := make([]int, 0, len(positions))
		for i := len(positions) - 1; i >= 0; i-- {
			pos := positions[i]
			if pos < 0 || pos >= len(s.items) {
				continue
			}
			if item := s.items[pos]; item != nil && strings.ToUpper(item.Method) == strings.ToUpper(method) {
				result = append(result, pos)
			}
		}
		return result
	}

	// 默认遍历全表（倒序）
	result := make([]int, 0, s.count)
	for i := 0; i < s.count; i++ {
		pos := (s.head - 1 - i + s.max) % s.max
		result = append(result, pos)
	}
	return result
}

func (s *RequestStore) addMethodIndex(method string, pos int) {
	if method == "" {
		return
	}
	key := strings.ToUpper(method)
	s.methodIndex[key] = append(s.methodIndex[key], pos)
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
