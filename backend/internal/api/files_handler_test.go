package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"droplite/internal/repository"
	"droplite/internal/service"
	"droplite/internal/storage"
)

type handlerRepo struct {
	createRecord *repository.FileRecord
	listResult   []repository.FileRecord
}

func (m *handlerRepo) Create(ctx context.Context, record *repository.FileRecord) (*repository.FileRecord, error) {
	m.createRecord = record
	return record, nil
}

func (m *handlerRepo) GetByID(ctx context.Context, id string) (*repository.FileRecord, error) {
	return nil, repository.ErrNotFound
}

func (m *handlerRepo) List(ctx context.Context, params repository.ListFilesParams) ([]repository.FileRecord, error) {
	return m.listResult, nil
}

func (m *handlerRepo) UpdateStatus(ctx context.Context, id string, status repository.FileStatus) error {
	return nil
}

type handlerWriter struct {
	calls int
}

func (w *handlerWriter) Write(ctx context.Context, key string, r io.Reader) (storage.Location, error) {
	_, _ = io.ReadAll(r)
	w.calls++
	return storage.Location{Path: key}, nil
}

func TestFileHandler_CreateFile(t *testing.T) {
	repo := &handlerRepo{}
	writer := &handlerWriter{}
	svc := service.NewFileService(repo, writer)
	handler := NewFileHandler(svc)

	payload := map[string]any{
		"original_name":  "hello.txt",
		"mime_type":      "text/plain",
		"size_bytes":     11,
		"storage_path":   "uploads/hello.txt",
		"content_base64": base64.StdEncoding.EncodeToString([]byte("hello world")),
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.CreateFile(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	if repo.createRecord == nil {
		t.Fatal("expected repository Create to be invoked")
	}
	if writer.calls != 1 {
		t.Fatalf("expected writer to be called once, got %d", writer.calls)
	}
}

func TestFileHandler_ListFiles(t *testing.T) {
	repo := &handlerRepo{
		listResult: []repository.FileRecord{{
			ID:           "1",
			OriginalName: "a",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}},
	}
	svc := service.NewFileService(repo, nil)
	handler := NewFileHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/files?limit=1", nil)
	rec := httptest.NewRecorder()

	handler.ListFiles(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Data []repository.FileRecord `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 record, got %d", len(resp.Data))
	}
}
