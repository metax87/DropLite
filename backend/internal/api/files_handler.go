package api

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"droplite/internal/repository"
	"droplite/internal/service"

	"github.com/go-chi/chi/v5"
)

// FileHandler 提供文件元数据相关的 HTTP 端点。
type FileHandler struct {
	service *service.FileService
}

func NewFileHandler(s *service.FileService) *FileHandler {
	return &FileHandler{service: s}
}

func (h *FileHandler) RegisterRoutes(r chi.Router) {
	r.Route("/files", func(r chi.Router) {
		r.Get("/", h.ListFiles)
		r.Post("/", h.CreateFile)
		r.Get("/{id}", h.GetFile)
		r.Get("/{id}/download", h.DownloadFile)
		r.Delete("/{id}", h.DeleteFile)
	})
}

type envelope struct {
	Data any `json:"data"`
}

type errorEnvelope struct {
	Error string `json:"error"`
}

const (
	maxUploadSizeBytes    int64 = 100 * 1024 * 1024 // 100MB
	multipartMemoryBudget int64 = 16 * 1024 * 1024
)

// CreateFile 接受 multipart/form-data 上传并登记文件元数据。
func (h *FileHandler) CreateFile(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		writeError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body is empty")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSizeBytes+multipartMemoryBudget)
	defer r.Body.Close()

	if err := r.ParseMultipartForm(multipartMemoryBudget); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid multipart form: %v", err))
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	sizeBytes, err := determineFileSize(file, header)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if sizeBytes <= 0 {
		writeError(w, http.StatusBadRequest, "file must not be empty")
		return
	}
	if sizeBytes > maxUploadSizeBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file exceeds size limit (100MB)")
		return
	}

	mimeType, err := resolveMimeType(header, file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := rewindFile(file); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to read uploaded file")
		return
	}

	originalName := header.Filename
	if override := strings.TrimSpace(r.FormValue("original_name")); override != "" {
		originalName = override
	}

	metadata, err := parseMetadataField(r.FormValue("metadata"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid metadata: "+err.Error())
		return
	}

	expiresAt, err := parseExpiresAt(r.FormValue("expires_at"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid expires_at: "+err.Error())
		return
	}

	record, err := h.service.RegisterFile(r.Context(), service.RegisterFileInput{
		OriginalName: originalName,
		MimeType:     mimeType,
		SizeBytes:    sizeBytes,
		Checksum:     optionalString(r.FormValue("checksum")),
		Metadata:     metadata,
		ExpiresAt:    expiresAt,
		Reader:       file,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, envelope{Data: record})
}

// ListFiles 返回文件集合。
func (h *FileHandler) ListFiles(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		writeError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	params := repository.ListFilesParams{}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			params.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			params.Offset = offset
		}
	}

	statuses := r.URL.Query()["status"]
	if len(statuses) == 0 {
		if combined := r.URL.Query().Get("statuses"); combined != "" {
			statuses = strings.Split(combined, ",")
		}
	}
	for _, raw := range statuses {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		params.Statuses = append(params.Statuses, repository.FileStatus(trimmed))
	}

	files, err := h.service.ListFiles(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, envelope{Data: files})
}

// DownloadFile 返回文件内容以供下载。
func (h *FileHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		writeError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "file id is required")
		return
	}

	file, err := h.service.GetFile(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	if file.Status != "stored" {
		writeError(w, http.StatusNotFound, "file not available for download")
		return
	}

	content, err := h.service.GetFileContent(r.Context(), file.StoragePath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}
	defer content.Close()

	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.OriginalName))
	w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))

	if _, err := io.Copy(w, content); err != nil {
		// 客户端可能已断开，无法再写入错误响应
		return
	}
}

// GetFile 返回单个文件的元数据。
func (h *FileHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		writeError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "file id is required")
		return
	}

	file, err := h.service.GetFile(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	writeJSON(w, http.StatusOK, envelope{Data: file})
}

// DeleteFile 软删除指定文件。
func (h *FileHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		writeError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "file id is required")
		return
	}

	if err := h.service.DeleteFile(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "file not found")
		return
	}

	writeJSON(w, http.StatusOK, envelope{Data: map[string]any{"id": id, "deleted": true}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorEnvelope{Error: message})
}

func determineFileSize(file multipart.File, header *multipart.FileHeader) (int64, error) {
	if header != nil && header.Size > 0 {
		return header.Size, nil
	}

	seeker, ok := file.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("cannot determine file size")
	}

	size, err := seeker.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("measure file: %w", err)
	}
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("rewind file: %w", err)
	}

	return size, nil
}

func resolveMimeType(header *multipart.FileHeader, file multipart.File) (string, error) {
	if header != nil {
		if value := header.Header.Get("Content-Type"); value != "" {
			return value, nil
		}
	}

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", fmt.Errorf("detect mime: %w", err)
	}

	if err := rewindFile(file); err != nil {
		return "", err
	}
	if n == 0 {
		return "application/octet-stream", nil
	}
	return http.DetectContentType(buf[:n]), nil
}

func rewindFile(file multipart.File) error {
	seeker, ok := file.(io.Seeker)
	if !ok {
		return fmt.Errorf("upload reader is not seekable")
	}
	_, err := seeker.Seek(0, io.SeekStart)
	return err
}

func parseMetadataField(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, err
	}
	return meta, nil
}

func parseExpiresAt(raw string) (*time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	ts, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &ts, nil
}

func optionalString(raw string) *string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	return &value
}
