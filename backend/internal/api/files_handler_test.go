package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
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

func (w *handlerWriter) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader([]byte("mock content"))), nil
}

func TestFileHandler_CreateFile(t *testing.T) {
	repo := &handlerRepo{}
	writer := &handlerWriter{}
	svc := service.NewFileService(repo, writer)
	handler := NewFileHandler(svc, 1024*1024*100)

	req := newMultipartRequest(t, map[string]string{
		"metadata": `{"env":"test"}`,
		"checksum": "abc123",
	}, "file", "hello.txt", []byte("hello world"))
	rec := httptest.NewRecorder()

	handler.CreateFile(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	if repo.createRecord == nil {
		t.Fatal("expected repository Create to be invoked")
	}
	if repo.createRecord.OriginalName != "hello.txt" {
		t.Fatalf("unexpected original name: %s", repo.createRecord.OriginalName)
	}
	if repo.createRecord.SizeBytes != 11 {
		t.Fatalf("unexpected size recorded: %d", repo.createRecord.SizeBytes)
	}
	if repo.createRecord.Metadata["env"] != "test" {
		t.Fatalf("expected metadata env, got %+v", repo.createRecord.Metadata)
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
	handler := NewFileHandler(svc, 1024*1024*100)

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

func newMultipartRequest(t *testing.T, fields map[string]string, fieldName, filename string, content []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form file: %v", err)
	}

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field %s: %v", k, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
