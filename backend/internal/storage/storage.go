package storage

import (
	"context"
	"io"
)

// Writer 定义对象存储写接口，支持流式写入。
type Writer interface {
	Write(ctx context.Context, key string, r io.Reader) (Location, error)
}

// Reader 定义对象存储读接口，支持流式读取。
type Reader interface {
	Read(ctx context.Context, key string) (io.ReadCloser, error)
}

// Storage 组合了读写能力的完整存储接口。
type Storage interface {
	Writer
	Reader
}

// Location 描述已经写入对象的可访问信息。
type Location struct {
	Path string
	URL  string
}
