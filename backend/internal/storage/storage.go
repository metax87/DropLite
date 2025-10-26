package storage

import (
	"context"
	"io"
)

// Writer 定义对象存储写接口，支持流式写入。
type Writer interface {
	Write(ctx context.Context, key string, r io.Reader) (Location, error)
}

// Location 描述已经写入对象的可访问信息。
type Location struct {
	Path string
	URL  string
}
