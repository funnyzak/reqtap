package request

import (
	"time"
)

// ReplayData represents a request replay record
type ReplayData struct {
	ID                string `json:"id"`
	OriginalRequestID string `json:"original_request_id"`
	Timestamp         time.Time `json:"timestamp"`
	Method            string `json:"method"`
	URL               string `json:"url"`
	Headers           map[string]string `json:"headers"`
	Body              []byte `json:"body"`
	StatusCode        int `json:"status_code"`
	ResponseBody      []byte `json:"response_body"`
	ResponseTimeMs    int64 `json:"response_time_ms"`
	Error             string `json:"error,omitempty"`
}

// ReplayRequest represents a replay request from API
type ReplayRequest struct {
	RequestID string            `json:"request_id"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	Query     string            `json:"query"`
	TargetURL string            `json:"target_url"`
}

// ReplayResponse represents a replay response to API
type ReplayResponse struct {
	ReplayID     string `json:"replay_id"`
	OriginalID   string `json:"original_id"`
	StatusCode   int    `json:"status_code"`
	ResponseBody string `json:"response_body"`
	ResponseTime int64  `json:"response_time_ms"`
	Error        string `json:"error,omitempty"`
}
