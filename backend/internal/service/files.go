package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"droplite/internal/repository"
	"droplite/internal/storage"
	"github.com/google/uuid"
)

// FileService 封装文件元数据的业务流程。
type FileService struct {
	repo  repository.FileRepository
	store storage.Writer
}

func NewFileService(repo repository.FileRepository, store storage.Writer) *FileService {
	return &FileService{repo: repo, store: store}
}

// RegisterFileInput 描述创建文件记录所需的信息。
type RegisterFileInput struct {
	OriginalName string
	MimeType     string
	SizeBytes    int64
	StoragePath  string
	Checksum     *string
	Metadata     map[string]any
	ExpiresAt    *time.Time
	Reader       io.Reader
}

// RegisterFile 创建新的文件元数据记录并写入存储。
func (s *FileService) RegisterFile(ctx context.Context, input RegisterFileInput) (*repository.FileRecord, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("file service not initialized")
	}
	if err := validateRegisterInput(input); err != nil {
		return nil, err
	}

	fileID := uuid.NewString()
	now := time.Now().UTC()

	if input.StoragePath == "" {
		input.StoragePath = defaultStoragePath(fileID, input.OriginalName, now)
	}
	record := &repository.FileRecord{
		ID:           fileID,
		OriginalName: input.OriginalName,
		MimeType:     input.MimeType,
		SizeBytes:    input.SizeBytes,
		StoragePath:  input.StoragePath,
		Checksum:     input.Checksum,
		Status:       repository.FileStatusPending,
		Metadata:     normalizeMetadata(input.Metadata),
		CreatedAt:    now,
		UpdatedAt:    now,
		ExpiresAt:    input.ExpiresAt,
	}

	if s.store != nil && input.Reader != nil {
		if _, err := s.store.Write(ctx, record.StoragePath, input.Reader); err != nil {
			return nil, fmt.Errorf("write storage: %w", err)
		}
		record.Status = repository.FileStatusStored
	}

	return s.repo.Create(ctx, record)
}

// ListFiles 以分页形式列出文件。
func (s *FileService) ListFiles(ctx context.Context, params repository.ListFilesParams) ([]repository.FileRecord, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("file service not initialized")
	}
	return s.repo.List(ctx, params)
}

func validateRegisterInput(input RegisterFileInput) error {
	switch {
	case input.OriginalName == "":
		return fmt.Errorf("original_name is required")
	case input.MimeType == "":
		return fmt.Errorf("mime_type is required")
	case input.SizeBytes <= 0:
		return fmt.Errorf("size_bytes must be positive")
	default:
		return nil
	}
}

func normalizeMetadata(meta map[string]any) map[string]any {
	if meta == nil {
		return map[string]any{}
	}
	return meta
}

func defaultStoragePath(fileID, originalName string, ts time.Time) string {
	safeName := sanitizeFilename(originalName)
	if safeName == "" {
		safeName = "file"
	}

	return filepath.ToSlash(filepath.Join(
		"uploads",
		fmt.Sprintf("%04d", ts.Year()),
		fmt.Sprintf("%02d", int(ts.Month())),
		fileID,
		safeName,
	))
}

var unsafeFilenameChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFilename(name string) string {
	if name == "" {
		return ""
	}
	base := filepath.Base(strings.TrimSpace(name))
	base = unsafeFilenameChars.ReplaceAllString(base, "_")
	base = strings.Trim(base, "._-")
	if base == "" {
		return ""
	}
	return base
}
