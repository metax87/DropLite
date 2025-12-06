package local

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"droplite/internal/storage"
)

// Writer 将文件写入本地文件系统。
type Writer struct {
	BaseDir string
	BaseURL string
}

func NewWriter(baseDir, baseURL string) *Writer {
	return &Writer{BaseDir: baseDir, BaseURL: baseURL}
}

func (w *Writer) Write(ctx context.Context, key string, r io.Reader) (storage.Location, error) {
	if w == nil {
		return storage.Location{}, fmt.Errorf("local writer uninitialized")
	}

	select {
	case <-ctx.Done():
		return storage.Location{}, ctx.Err()
	default:
	}

	targetPath := filepath.Join(w.BaseDir, filepath.Clean(key))
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return storage.Location{}, fmt.Errorf("ensure dir: %w", err)
	}

	tempPath := targetPath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return storage.Location{}, fmt.Errorf("create temp file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		file.Close()
		os.Remove(tempPath)
		return storage.Location{}, fmt.Errorf("write file: %w", err)
	}

	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(tempPath)
		return storage.Location{}, fmt.Errorf("sync file: %w", err)
	}

	if err := file.Close(); err != nil {
		return storage.Location{}, fmt.Errorf("close file: %w", err)
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		return storage.Location{}, fmt.Errorf("rename temp file: %w", err)
	}

	loc := storage.Location{Path: targetPath}
	if w.BaseURL != "" {
		u, err := url.JoinPath(w.BaseURL, filepath.ToSlash(key))
		if err == nil {
			loc.URL = u
		}
	}

	return loc, nil
}

// Read 打开并返回指定 key 对应的文件内容。
func (w *Writer) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	if w == nil {
		return nil, fmt.Errorf("local writer uninitialized")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	targetPath := filepath.Join(w.BaseDir, filepath.Clean(key))
	file, err := os.Open(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("open file: %w", err)
	}

	return file, nil
}
