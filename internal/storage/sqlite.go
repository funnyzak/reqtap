package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/funnyzak/reqtap/internal/config"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/pkg/request"

	_ "modernc.org/sqlite"
)

const (
	sqliteDriverName = "sqlite"
)

type sqliteStore struct {
	db  *sql.DB
	cfg *config.StorageConfig
	log logger.Logger
}

func newSQLiteStore(cfg *config.StorageConfig, log logger.Logger) (Store, error) {
	path := cfg.Path
	if path == "" {
		return nil, fmt.Errorf("sqlite path cannot be empty")
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve sqlite path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, fmt.Errorf("prepare sqlite directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_foreign_keys=on", filepath.ToSlash(absPath))
	db, err := sql.Open(sqliteDriverName, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(8)
	db.SetMaxOpenConns(8)
	db.SetConnMaxLifetime(0)

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA temp_store=MEMORY;",
		"PRAGMA mmap_size=268435456;",
	}
	for _, stmt := range pragmas {
		if _, err := db.Exec(stmt); err != nil {
			db.Close()
			return nil, fmt.Errorf("apply pragma %s: %w", stmt, err)
		}
	}

	store := &sqliteStore{db: db, cfg: cfg, log: log}
	if err := store.initSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *sqliteStore) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS requests (
    id TEXT PRIMARY KEY,
    timestamp_ns INTEGER NOT NULL,
    method TEXT NOT NULL,
    proto TEXT,
    path TEXT,
    query TEXT,
    remote_addr TEXT,
    user_agent TEXT,
    headers_json TEXT,
    body BLOB,
    content_type TEXT,
    content_length INTEGER,
    is_binary INTEGER,
    size INTEGER,
    mock_rule TEXT,
    mock_status INTEGER
);
CREATE INDEX IF NOT EXISTS idx_requests_ts ON requests(timestamp_ns DESC);
CREATE INDEX IF NOT EXISTS idx_requests_method_ts ON requests(method, timestamp_ns DESC);

CREATE TABLE IF NOT EXISTS replays (
    id TEXT PRIMARY KEY,
    original_request_id TEXT NOT NULL,
    timestamp_ns INTEGER NOT NULL,
    method TEXT NOT NULL,
    url TEXT NOT NULL,
    headers_json TEXT,
    body BLOB,
    status_code INTEGER,
    response_body BLOB,
    response_time_ms INTEGER,
    error TEXT,
    FOREIGN KEY (original_request_id) REFERENCES requests(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_replays_ts ON replays(timestamp_ns DESC);
CREATE INDEX IF NOT EXISTS idx_replays_original ON replays(original_request_id);
`
	_, err := s.db.Exec(schema)
	return err
}

func (s *sqliteStore) Record(data *request.RequestData) (*StoredRequest, error) {
	if data == nil {
		return nil, fmt.Errorf("request data is nil")
	}
	if strings.TrimSpace(data.ID) == "" {
		data.ID = fmt.Sprintf("REQ-%d", time.Now().UnixNano())
	}
	ctx := context.Background()
	ts := data.Timestamp.UTC()
	if ts.IsZero() {
		ts = time.Now().UTC()
		data.Timestamp = ts
	} else {
		data.Timestamp = ts
	}
	if data.Size == 0 {
		data.Size = int64(len(data.Body))
	}
	headers := data.Headers
	if headers == nil {
		headers = http.Header{}
	}
	headersJSON, err := json.Marshal(headers)
	if err != nil {
		return nil, fmt.Errorf("marshal headers: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	insertSQL := `INSERT INTO requests (
        id, timestamp_ns, method, proto, path, query, remote_addr, user_agent,
        headers_json, body, content_type, content_length, is_binary, size,
        mock_rule, mock_status
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.ExecContext(ctx, insertSQL,
		data.ID,
		ts.UnixNano(),
		data.Method,
		data.Proto,
		data.Path,
		data.Query,
		data.RemoteAddr,
		data.UserAgent,
		string(headersJSON),
		data.Body,
		data.ContentType,
		data.ContentLength,
		boolToInt(data.IsBinary),
		data.Size,
		data.MockResponse.Rule,
		data.MockResponse.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("insert request: %w", err)
	}

	if err = s.prune(ctx, tx); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &StoredRequest{ID: data.ID, RequestData: data}, nil
}

func (s *sqliteStore) prune(ctx context.Context, tx *sql.Tx) error {
	if s.cfg.Retention > 0 {
		cutoff := time.Now().Add(-s.cfg.Retention).UTC().UnixNano()
		if _, err := tx.ExecContext(ctx, "DELETE FROM requests WHERE timestamp_ns < ?", cutoff); err != nil {
			return fmt.Errorf("prune by retention: %w", err)
		}
	}
	if s.cfg.MaxRecords > 0 {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM requests").Scan(&count); err != nil {
			return fmt.Errorf("count records: %w", err)
		}
		if count > s.cfg.MaxRecords {
			excess := count - s.cfg.MaxRecords
			if excess < 0 {
				excess = 0
			}
			if excess > 0 {
				if _, err := tx.ExecContext(ctx, "DELETE FROM requests WHERE id IN (SELECT id FROM requests ORDER BY timestamp_ns ASC LIMIT ?)", excess); err != nil {
					return fmt.Errorf("prune max records: %w", err)
				}
			}
		}
	}
	return nil
}

func (s *sqliteStore) List(opts ListOptions) ([]*StoredRequest, int, error) {
	ctx := context.Background()
	where, args := buildFilters(opts)

	countQuery := fmt.Sprintf("SELECT COUNT(1) FROM requests %s", where)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	queryBuilder := strings.Builder{}
	queryBuilder.WriteString("SELECT id, timestamp_ns, method, proto, path, query, remote_addr, user_agent, headers_json, body, content_type, content_length, is_binary, size, mock_rule, mock_status FROM requests ")
	queryBuilder.WriteString(where)
	queryBuilder.WriteString(" ORDER BY timestamp_ns DESC")

	limit := opts.Limit
	offset := opts.Offset
	var listArgs []interface{}
	listArgs = append(listArgs, args...)
	if limit > 0 {
		if offset < 0 {
			offset = 0
		}
		queryBuilder.WriteString(" LIMIT ? OFFSET ?")
		listArgs = append(listArgs, limit, offset)
	}

	rows, err := s.db.QueryContext(ctx, queryBuilder.String(), listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*StoredRequest
	for rows.Next() {
		record, err := scanStoredRequest(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return result, total, nil
}

func (s *sqliteStore) Iterate(opts ListOptions, fn func(*StoredRequest) bool) error {
	ctx := context.Background()
	where, args := buildFilters(opts)

	query := strings.Builder{}
	query.WriteString("SELECT id, timestamp_ns, method, proto, path, query, remote_addr, user_agent, headers_json, body, content_type, content_length, is_binary, size, mock_rule, mock_status FROM requests ")
	query.WriteString(where)
	query.WriteString(" ORDER BY timestamp_ns DESC")

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		record, err := scanStoredRequest(rows)
		if err != nil {
			return err
		}
		if !fn(record) {
			break
		}
	}
	return rows.Err()
}

func (s *sqliteStore) Snapshot() ([]*StoredRequest, error) {
	var records []*StoredRequest
	err := s.Iterate(ListOptions{}, func(item *StoredRequest) bool {
		records = append(records, item)
		return true
	})
	return records, err
}

func (s *sqliteStore) Get(id string) (*StoredRequest, error) {
	ctx := context.Background()
	row := s.db.QueryRowContext(ctx, "SELECT id, timestamp_ns, method, proto, path, query, remote_addr, user_agent, headers_json, body, content_type, content_length, is_binary, size, mock_rule, mock_status FROM requests WHERE id = ?", id)
	record, err := scanStoredRequest(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (s *sqliteStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func scanStoredRequest(scanner interface {
	Scan(dest ...interface{}) error
}) (*StoredRequest, error) {
	var (
		id          string
		ts          int64
		method      string
		proto       sql.NullString
		path        sql.NullString
		query       sql.NullString
		remote      sql.NullString
		userAgent   sql.NullString
		headersJSON sql.NullString
		body        []byte
		contentType sql.NullString
		contentLen  sql.NullInt64
		isBinary    int64
		size        sql.NullInt64
		mockRule    sql.NullString
		mockStatus  sql.NullInt64
	)

	if err := scanner.Scan(
		&id,
		&ts,
		&method,
		&proto,
		&path,
		&query,
		&remote,
		&userAgent,
		&headersJSON,
		&body,
		&contentType,
		&contentLen,
		&isBinary,
		&size,
		&mockRule,
		&mockStatus,
	); err != nil {
		return nil, err
	}

	header := http.Header{}
	if headersJSON.Valid && headersJSON.String != "" {
		if err := json.Unmarshal([]byte(headersJSON.String), &header); err != nil {
			header = http.Header{}
		}
	}

	data := &request.RequestData{
		ID:            id,
		Timestamp:     time.Unix(0, ts).UTC(),
		Method:        method,
		Proto:         proto.String,
		Path:          path.String,
		Query:         query.String,
		RemoteAddr:    remote.String,
		UserAgent:     userAgent.String,
		Headers:       header,
		Body:          append([]byte(nil), body...),
		ContentType:   contentType.String,
		ContentLength: contentLen.Int64,
		IsBinary:      isBinary == 1,
		Size:          size.Int64,
		MockResponse: request.MockResponse{
			Rule:   mockRule.String,
			Status: int(mockStatus.Int64),
		},
	}
	if data.Size == 0 {
		data.Size = int64(len(body))
	}
	return &StoredRequest{ID: id, RequestData: data}, nil
}

func buildFilters(opts ListOptions) (string, []interface{}) {
	var clauses []string
	var args []interface{}

	if method := strings.TrimSpace(opts.Method); method != "" {
		clauses = append(clauses, "UPPER(method) = UPPER(?)")
		args = append(args, method)
	}

	if search := strings.TrimSpace(strings.ToLower(opts.Search)); search != "" {
		like := fmt.Sprintf("%%%s%%", search)
		clauses = append(clauses, "(LOWER(path) LIKE ? OR LOWER(query) LIKE ? OR LOWER(remote_addr) LIKE ? OR LOWER(user_agent) LIKE ? OR LOWER(headers_json) LIKE ?)")
		args = append(args, like, like, like, like, like)
	}

	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// RecordReplay stores a replay record
func (s *sqliteStore) RecordReplay(data *request.ReplayData) (*StoredReplay, error) {
	if data == nil {
		return nil, fmt.Errorf("replay data is nil")
	}
	if strings.TrimSpace(data.ID) == "" {
		data.ID = fmt.Sprintf("RPL-%d", time.Now().UnixNano())
	}

	ctx := context.Background()
	ts := data.Timestamp.UTC()
	if ts.IsZero() {
		ts = time.Now().UTC()
		data.Timestamp = ts
	}

	headersJSON, err := json.Marshal(data.Headers)
	if err != nil {
		return nil, fmt.Errorf("marshal headers: %w", err)
	}

	insertSQL := `INSERT INTO replays (
		id, original_request_id, timestamp_ns, method, url,
		headers_json, body, status_code, response_body, response_time_ms, error
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.ExecContext(ctx, insertSQL,
		data.ID,
		data.OriginalRequestID,
		ts.UnixNano(),
		data.Method,
		data.URL,
		string(headersJSON),
		data.Body,
		data.StatusCode,
		data.ResponseBody,
		data.ResponseTimeMs,
		data.Error,
	)
	if err != nil {
		return nil, fmt.Errorf("insert replay: %w", err)
	}

	return &StoredReplay{ReplayData: data}, nil
}

// GetReplays retrieves all replays for a specific request
func (s *sqliteStore) GetReplays(originalRequestID string) ([]*StoredReplay, error) {
	ctx := context.Background()
	query := `SELECT id, original_request_id, timestamp_ns, method, url,
		headers_json, body, status_code, response_body, response_time_ms, error
		FROM replays WHERE original_request_id = ? ORDER BY timestamp_ns DESC`

	rows, err := s.db.QueryContext(ctx, query, originalRequestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*StoredReplay
	for rows.Next() {
		replay, err := scanStoredReplay(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, replay)
	}

	return result, rows.Err()
}

func scanStoredReplay(scanner interface {
	Scan(dest ...interface{}) error
}) (*StoredReplay, error) {
	var (
		id                string
		originalRequestID string
		ts                int64
		method            string
		url               string
		headersJSON       sql.NullString
		body              []byte
		statusCode        sql.NullInt64
		responseBody      []byte
		responseTimeMs    sql.NullInt64
		errorMsg          sql.NullString
	)

	if err := scanner.Scan(
		&id,
		&originalRequestID,
		&ts,
		&method,
		&url,
		&headersJSON,
		&body,
		&statusCode,
		&responseBody,
		&responseTimeMs,
		&errorMsg,
	); err != nil {
		return nil, err
	}

	headers := make(map[string]string)
	if headersJSON.Valid && headersJSON.String != "" {
		if err := json.Unmarshal([]byte(headersJSON.String), &headers); err != nil {
			headers = make(map[string]string)
		}
	}

	data := &request.ReplayData{
		ID:                id,
		OriginalRequestID: originalRequestID,
		Timestamp:         time.Unix(0, ts).UTC(),
		Method:            method,
		URL:               url,
		Headers:           headers,
		Body:              append([]byte(nil), body...),
		StatusCode:        int(statusCode.Int64),
		ResponseBody:      append([]byte(nil), responseBody...),
		ResponseTimeMs:    responseTimeMs.Int64,
		Error:             errorMsg.String,
	}

	return &StoredReplay{ReplayData: data}, nil
}
