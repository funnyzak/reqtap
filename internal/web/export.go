package web

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ExportRequests serializes stored requests into the desired format.
func ExportRequests(data []*StoredRequest, format string) ([]byte, string, string, error) {
	switch strings.ToLower(format) {
	case "json":
		buf, err := json.MarshalIndent(data, "", "  ")
		return buf, "application/json", "json", err
	case "csv":
		return exportCSV(data)
	default:
		return nil, "", "", fmt.Errorf("unsupported export format: %s", format)
	}
}

func exportCSV(data []*StoredRequest) ([]byte, string, string, error) {
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)

	headers := []string{
		"id", "timestamp", "method", "path", "query", "remote_addr",
		"user_agent", "content_type", "content_length", "is_binary", "headers", "body_base64",
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", "", err
	}

	for _, item := range data {
		headersJSON, _ := json.Marshal(item.Headers)
		line := []string{
			item.ID,
			item.Timestamp.Format(time.RFC3339),
			item.Method,
			item.Path,
			item.Query,
			item.RemoteAddr,
			item.UserAgent,
			item.ContentType,
			fmt.Sprintf("%d", item.ContentLength),
			fmt.Sprintf("%t", item.IsBinary),
			string(headersJSON),
			base64.StdEncoding.EncodeToString(item.Body),
		}
		if err := writer.Write(line); err != nil {
			return nil, "", "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", "", err
	}

	return buf.Bytes(), "text/csv", "csv", nil
}

// AllowedFormats normalizes configured export formats.
func AllowedFormats(formats []string) []string {
	set := make(map[string]struct{})
	for _, f := range formats {
		f = strings.ToLower(strings.TrimSpace(f))
		if f == "" {
			continue
		}
		set[f] = struct{}{}
	}

	result := make([]string, 0, len(set))
	for f := range set {
		result = append(result, f)
	}
	sort.Strings(result)
	return result
}
