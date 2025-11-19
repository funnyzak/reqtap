package web

import "github.com/funnyzak/reqtap/internal/storage"

// StoredRequest 是 storage.StoredRequest 的别名，方便 Web 模块复用。
type StoredRequest = storage.StoredRequest

// ListOptions 是 storage.ListOptions 的别名，兼容现有调用。
type ListOptions = storage.ListOptions
