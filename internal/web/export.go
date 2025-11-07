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
	case "text", "txt":
		return exportText(data)
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

func exportText(data []*StoredRequest) ([]byte, string, string, error) {
	var builder strings.Builder
	for i, item := range data {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(renderPlainRequest(item))
	}

	return []byte(builder.String()), "text/plain; charset=utf-8", "txt", nil
}

func renderPlainRequest(item *StoredRequest) string {
	if item == nil {
		return ""
	}
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("# Request %s @ %s\n", item.ID, item.Timestamp.Format(time.RFC3339)))
	if item.RemoteAddr != "" {
		builder.WriteString(fmt.Sprintf("# Remote: %s\n", item.RemoteAddr))
	}
	bodySize := len(item.Body)
	if bodySize > 0 {
		builder.WriteString(fmt.Sprintf("# Body-Size: %d bytes\n", bodySize))
	}
	builder.WriteString("\n")
	builder.WriteString(buildHTTPRequestMessage(item))
	return builder.String()
}

func buildHTTPRequestMessage(item *StoredRequest) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s %s HTTP/1.1\r\n", strings.ToUpper(item.Method), composeFullPath(item)))
	for _, key := range sortedHeaderKeys(item.Headers) {
		values := item.Headers[key]
		for _, value := range values {
			builder.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
		}
	}
	builder.WriteString("\r\n")
	if len(item.Body) == 0 {
		builder.WriteString("(empty body)")
	} else if item.IsBinary {
		builder.WriteString(fmt.Sprintf("[binary payload omitted, size=%d bytes]", len(item.Body)))
	} else {
		builder.Write(item.Body)
	}
	builder.WriteString("\r\n")
	return builder.String()
}

func composeFullPath(item *StoredRequest) string {
	if item == nil {
		return "/"
	}
	path := item.Path
	if path == "" {
		path = "/"
	}
	if item.Query != "" {
		path = path + "?" + item.Query
	}
	return path
}

func sortedHeaderKeys(headers map[string][]string) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.ToLower(keys[i]) < strings.ToLower(keys[j])
	})
	return keys
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
