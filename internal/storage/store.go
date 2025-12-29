package storage

import (
	"errors"

	"github.com/funnyzak/reqtap/internal/config"
	"github.com/funnyzak/reqtap/internal/logger"
	"github.com/funnyzak/reqtap/pkg/request"
)

// ErrUnsupportedDriver indicates the configured driver is not available.
var ErrUnsupportedDriver = errors.New("unsupported storage driver")

// ListOptions controls filtering and pagination when fetching requests.
type ListOptions struct {
	Search string
	Method string
	Limit  int
	Offset int
}

// StoredRequest wraps RequestData with its persisted identifier.
type StoredRequest struct {
	ID string `json:"id"`
	*request.RequestData
}

// StoredReplay wraps ReplayData with storage metadata
type StoredReplay struct {
	*request.ReplayData
}

// Store defines the persistence contract for captured requests.
type Store interface {
	Record(*request.RequestData) (*StoredRequest, error)
	List(ListOptions) ([]*StoredRequest, int, error)
	Iterate(ListOptions, func(*StoredRequest) bool) error
	Snapshot() ([]*StoredRequest, error)
	Get(string) (*StoredRequest, error)

	// Replay related methods
	RecordReplay(*request.ReplayData) (*StoredReplay, error)
	GetReplays(originalRequestID string) ([]*StoredReplay, error)

	Close() error
}

// New instantiates a Store based on configuration.
func New(cfg *config.StorageConfig, log logger.Logger) (Store, error) {
	if cfg == nil {
		return nil, errors.New("storage config is nil")
	}
	switch driver := cfg.Driver; driver {
	case "", "sqlite", "sqlite3":
		return newSQLiteStore(cfg, log)
	default:
		return nil, ErrUnsupportedDriver
	}
}
