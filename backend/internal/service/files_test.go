package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"droplite/internal/repository"
	"droplite/internal/storage"
)

type mockFileRepo struct {
	createRecord *repository.FileRecord
	createErr    error
	listParams   repository.ListFilesParams
	listResult   []repository.FileRecord
	listErr      error
}

func (m *mockFileRepo) Create(ctx context.Context, record *repository.FileRecord) (*repository.FileRecord, error) {
	m.createRecord = record
	if m.createErr != nil {
		return nil, m.createErr
	}
	return record, nil
}

func (m *mockFileRepo) GetByID(ctx context.Context, id string) (*repository.FileRecord, error) {
	return nil, repository.ErrNotFound
}

func (m *mockFileRepo) List(ctx context.Context, params repository.ListFilesParams) ([]repository.FileRecord, error) {
	m.listParams = params
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockFileRepo) UpdateStatus(ctx context.Context, id string, status repository.FileStatus) error {
	return nil
}

type mockWriter struct {
	key  string
	data []byte
	err  error
}

func (w *mockWriter) Write(ctx context.Context, key string, r io.Reader) (storage.Location, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return storage.Location{}, err
	}
	w.key = key
	w.data = body
	if w.err != nil {
		return storage.Location{}, w.err
	}
	return storage.Location{Path: key}, nil
}

func TestFileService_RegisterFile_WritesStorageAndRepository(t *testing.T) {
	repo := &mockFileRepo{}
	writer := &mockWriter{}
	svc := NewFileService(repo, writer)

	payload := []byte("hello world")
	input := RegisterFileInput{
		OriginalName: "greeting.txt",
		MimeType:     "text/plain",
		SizeBytes:    int64(len(payload)),
		StoragePath:  "uploads/greeting.txt",
		Reader:       bytes.NewReader(payload),
	}

	record, err := svc.RegisterFile(context.Background(), input)
	if err != nil {
		t.Fatalf("RegisterFile returned error: %v", err)
	}
	if record == nil {
		t.Fatalf("expected record, got nil")
	}
	if repo.createRecord == nil {
		t.Fatalf("repository Create was not called")
	}
	if writer.key != input.StoragePath {
		t.Fatalf("expected writer key %s, got %s", input.StoragePath, writer.key)
	}
	if string(writer.data) != string(payload) {
		t.Fatalf("expected writer data %q, got %q", payload, writer.data)
	}
}

func TestFileService_RegisterFile_Validation(t *testing.T) {
	svc := NewFileService(&mockFileRepo{}, &mockWriter{})
	_, err := svc.RegisterFile(context.Background(), RegisterFileInput{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestFileService_ListFiles_DelegatesToRepo(t *testing.T) {
	repo := &mockFileRepo{
		listResult: []repository.FileRecord{{ID: "1", OriginalName: "a"}},
	}
	svc := NewFileService(repo, nil)

	params := repository.ListFilesParams{Limit: 5}
	records, err := svc.ListFiles(context.Background(), params)
	if err != nil {
		t.Fatalf("ListFiles returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if repo.listParams.Limit != params.Limit {
		t.Fatalf("repository received wrong params: %+v", repo.listParams)
	}
}

func TestFileService_RegisterFile_StorageError(t *testing.T) {
	repo := &mockFileRepo{}
	writer := &mockWriter{err: errors.New("boom")}
	svc := NewFileService(repo, writer)

	_, err := svc.RegisterFile(context.Background(), RegisterFileInput{
		OriginalName: "boom.txt",
		MimeType:     "text/plain",
		SizeBytes:    4,
		StoragePath:  "uploads/boom.txt",
		Reader:       bytes.NewReader([]byte("data")),
	})
	if err == nil {
		t.Fatal("expected storage error, got nil")
	}
	if repo.createRecord != nil {
		t.Fatal("repository should not be called when storage fails")
	}
}

func TestFileService_RegisterFile_SkipsStorageWhenNilReader(t *testing.T) {
	repo := &mockFileRepo{}
	writer := &mockWriter{}
	svc := NewFileService(repo, writer)

	record, err := svc.RegisterFile(context.Background(), RegisterFileInput{
		OriginalName: "meta-only",
		MimeType:     "text/plain",
		SizeBytes:    1,
		StoragePath:  "uploads/meta.txt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record == nil {
		t.Fatalf("expected record")
	}
	if writer.key != "" {
		t.Fatalf("writer should not run when reader is nil")
	}
}
