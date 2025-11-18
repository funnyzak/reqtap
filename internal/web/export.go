package web

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// RequestIterator 用于按需遍历请求
type RequestIterator func(func(*StoredRequest) bool)

// ExportRequests serializes stored requests into the desired format.
func ExportRequests(data []*StoredRequest, format string) ([]byte, string, string, error) {
	iter := func(yield func(*StoredRequest) bool) {
		for i := 0; i < len(data); i++ {
			if !yield(data[i]) {
				return
			}
		}
	}
	buf := &bytes.Buffer{}
	contentType, ext, err := StreamExport(buf, iter, format)
	return buf.Bytes(), contentType, ext, err
}

// StreamExport 以流式方式导出，避免大数据加载内存
func StreamExport(w io.Writer, iter RequestIterator, format string) (string, string, error) {
	contentType, ext, err := describeFormat(format)
	if err != nil {
		return "", "", err
	}

	var streamErr error
	switch strings.ToLower(format) {
	case "json":
		streamErr = streamJSON(w, iter)
	case "csv":
		streamErr = streamCSV(w, iter)
	case "text", "txt":
		streamErr = streamText(w, iter)
	}
	return contentType, ext, streamErr
}

func describeFormat(format string) (string, string, error) {
	switch strings.ToLower(format) {
	case "json":
		return "application/json", "json", nil
	case "csv":
		return "text/csv", "csv", nil
	case "text", "txt":
		return "text/plain; charset=utf-8", "txt", nil
	default:
		return "", "", fmt.Errorf("unsupported export format: %s", format)
	}
}

func streamJSON(w io.Writer, iter RequestIterator) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	if _, err := bw.WriteString("["); err != nil {
		return err
	}
	first := true
	var marshalErr error
	iter(func(item *StoredRequest) bool {
		if marshalErr != nil {
			return false
		}
		b, err := json.Marshal(item)
		if err != nil {
			marshalErr = err
			return false
		}
		if !first {
			if _, marshalErr = bw.WriteString(","); marshalErr != nil {
				return false
			}
		}
		first = false
		if _, marshalErr = bw.Write(b); marshalErr != nil {
			return false
		}
		return true
	})
	if marshalErr != nil {
		return marshalErr
	}
	_, err := bw.WriteString("]")
	return err
}

func streamCSV(w io.Writer, iter RequestIterator) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	csvWriter := csv.NewWriter(bw)
	headers := []string{
		"id", "timestamp", "method", "path", "query", "remote_addr",
		"user_agent", "content_type", "content_length", "is_binary", "headers", "body_base64",
	}
	if err := csvWriter.Write(headers); err != nil {
		return err
	}

	var writeErr error
	iter(func(item *StoredRequest) bool {
		if writeErr != nil {
			return false
		}
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
		writeErr = csvWriter.Write(line)
		return writeErr == nil
	})

	csvWriter.Flush()
	if writeErr != nil {
		return writeErr
	}
	return csvWriter.Error()
}

func streamText(w io.Writer, iter RequestIterator) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	first := true
	var writeErr error
	iter(func(item *StoredRequest) bool {
		if writeErr != nil {
			return false
		}
		if !first {
			if _, writeErr = bw.WriteString("\n\n"); writeErr != nil {
				return false
			}
		}
		first = false
		_, writeErr = bw.WriteString(renderPlainRequest(item))
		return writeErr == nil
	})
	return writeErr
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
