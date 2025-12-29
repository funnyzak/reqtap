package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/funnyzak/reqtap/pkg/request"
)

// handleReplay handles request replay
func (s *Service) handleReplay(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		s.logger.Error("Storage not configured for web service")
		return
	}

	// Parse replay request
	var req request.ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		s.logger.Error("Failed to decode replay request", "error", err)
		return
	}

	// Validate request
	if req.RequestID == "" {
		http.Error(w, "request_id is required", http.StatusBadRequest)
		return
	}
	if req.TargetURL == "" {
		http.Error(w, "target_url is required", http.StatusBadRequest)
		return
	}

	// Get original request
	originalReq, err := s.store.Get(req.RequestID)
	if err != nil {
		s.logger.Error("Failed to get original request", "request_id", req.RequestID, "error", err)
		http.Error(w, "Failed to retrieve original request", http.StatusInternalServerError)
		return
	}
	if originalReq == nil {
		http.Error(w, "Original request not found", http.StatusNotFound)
		return
	}

	// Use provided values or fallback to original
	method := req.Method
	if method == "" {
		method = originalReq.Method
	}

	headers := req.Headers
	if headers == nil {
		headers = make(map[string]string)
		for k, v := range originalReq.Headers {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
	}

	body := []byte(req.Body)
	if len(body) == 0 {
		body = originalReq.Body
	}

	// Build target URL with query
	targetURL := req.TargetURL
	if req.Query != "" {
		if strings.Contains(targetURL, "?") {
			targetURL += "&" + req.Query
		} else {
			targetURL += "?" + req.Query
		}
	}

	// Perform replay
	replayData, err := s.performReplay(r.Context(), method, targetURL, headers, body, req.RequestID)
	if err != nil {
		s.logger.Error("Failed to perform replay", "error", err)
		http.Error(w, "Failed to replay request", http.StatusInternalServerError)
		return
	}

	// Store replay result
	stored, err := s.store.RecordReplay(replayData)
	if err != nil {
		s.logger.Error("Failed to store replay", "error", err)
		// Continue even if storage fails
	}

	// Build response
	response := request.ReplayResponse{
		ReplayID:     replayData.ID,
		OriginalID:   req.RequestID,
		StatusCode:   replayData.StatusCode,
		ResponseBody: string(replayData.ResponseBody),
		ResponseTime: replayData.ResponseTimeMs,
		Error:        replayData.Error,
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	s.logger.Info("Request replayed",
		"replay_id", replayData.ID,
		"original_id", req.RequestID,
		"target_url", targetURL,
		"status_code", replayData.StatusCode,
		"response_time_ms", replayData.ResponseTimeMs)

	// Notify if stored successfully
	if stored != nil {
		// Could broadcast to websocket here if needed
	}
}

// handleGetReplays returns all replays for a request
func (s *Service) handleGetReplays(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		http.Error(w, "storage unavailable", http.StatusServiceUnavailable)
		return
	}

	requestID := r.URL.Query().Get("request_id")
	if requestID == "" {
		http.Error(w, "request_id parameter is required", http.StatusBadRequest)
		return
	}

	replays, err := s.store.GetReplays(requestID)
	if err != nil {
		s.logger.Error("Failed to get replays", "request_id", requestID, "error", err)
		http.Error(w, "Failed to retrieve replays", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"request_id": requestID,
		"replays":    replays,
		"total":      len(replays),
	})
}

// performReplay executes the actual HTTP request
func (s *Service) performReplay(ctx context.Context, method, targetURL string, headers map[string]string, body []byte, originalRequestID string) (*request.ReplayData, error) {
	startTime := time.Now()

	replayData := &request.ReplayData{
		ID:                fmt.Sprintf("RPL-%d", time.Now().UnixNano()),
		OriginalRequestID: originalRequestID,
		Timestamp:         startTime,
		Method:            method,
		URL:               targetURL,
		Headers:           headers,
		Body:              body,
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, targetURL, bytes.NewReader(body))
	if err != nil {
		replayData.Error = fmt.Sprintf("Failed to create request: %v", err)
		return replayData, nil // Return data with error, don't fail
	}

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Add replay tracking header
	req.Header.Set("X-ReqTap-Replay", "true")
	req.Header.Set("X-ReqTap-Replay-ID", replayData.ID)
	req.Header.Set("X-ReqTap-Original-ID", originalRequestID)

	// Perform request
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		replayData.Error = fmt.Sprintf("Request failed: %v", err)
		replayData.ResponseTimeMs = time.Since(startTime).Milliseconds()
		return replayData, nil
	}
	defer resp.Body.Close()

	// Read response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		replayData.Error = fmt.Sprintf("Failed to read response: %v", err)
		replayData.StatusCode = resp.StatusCode
		replayData.ResponseTimeMs = time.Since(startTime).Milliseconds()
		return replayData, nil
	}

	// Set response data
	replayData.StatusCode = resp.StatusCode
	replayData.ResponseBody = responseBody
	replayData.ResponseTimeMs = time.Since(startTime).Milliseconds()

	return replayData, nil
}

// parseURL safely parses a URL
func parseURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	return u, nil
}
