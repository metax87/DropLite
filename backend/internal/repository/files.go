package repository

import (
	"context"
	"time"
)

// FileStatus 描述上传生命周期。
type FileStatus string

const (
	FileStatusPending FileStatus = "pending"
	FileStatusStored  FileStatus = "stored"
	FileStatusFailed  FileStatus = "failed"
	FileStatusDeleted FileStatus = "deleted"
)

// FileRecord 代表数据库中的文件元数据。
type FileRecord struct {
	ID           string         `json:"id"`
	OriginalName string         `json:"original_name"`
	MimeType     string         `json:"mime_type"`
	SizeBytes    int64          `json:"size_bytes"`
	StoragePath  string         `json:"storage_path"`
	Checksum     *string        `json:"checksum,omitempty"`
	Status       FileStatus     `json:"status"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	ExpiresAt    *time.Time     `json:"expires_at,omitempty"`
}

// ListFilesParams 用于分页检索文件。
type ListFilesParams struct {
	Statuses []FileStatus
	Limit    int
	Offset   int
}

// FileRepository 统一文件元数据持久层接口。
type FileRepository interface {
	Create(ctx context.Context, record *FileRecord) (*FileRecord, error)
	GetByID(ctx context.Context, id string) (*FileRecord, error)
	List(ctx context.Context, params ListFilesParams) ([]FileRecord, error)
	UpdateStatus(ctx context.Context, id string, status FileStatus) error
}
