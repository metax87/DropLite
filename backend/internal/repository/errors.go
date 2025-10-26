package repository

import "errors"

// ErrNotFound 表示目标记录不存在。
var ErrNotFound = errors.New("repository: record not found")
