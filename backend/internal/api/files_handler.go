package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
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
	})
}

type createFileRequest struct {
	OriginalName  string         `json:"original_name"`
	MimeType      string         `json:"mime_type"`
	SizeBytes     int64          `json:"size_bytes"`
	StoragePath   string         `json:"storage_path"`
	Checksum      *string        `json:"checksum"`
	Metadata      map[string]any `json:"metadata"`
	ExpiresAt     *time.Time     `json:"expires_at"`
	ContentBase64 string         `json:"content_base64"`
}

type envelope struct {
	Data any `json:"data"`
}

type errorEnvelope struct {
	Error string `json:"error"`
}

// CreateFile 仅登记元数据，实际文件写入将由后续存储模块负责。
func (h *FileHandler) CreateFile(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		writeError(w, http.StatusInternalServerError, "handler not initialized")
		return
	}
	defer r.Body.Close()

	var req createFileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var reader *bytes.Reader
	if req.ContentBase64 != "" {
		payload, err := base64.StdEncoding.DecodeString(req.ContentBase64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid content_base64")
			return
		}
		reader = bytes.NewReader(payload)
	}

	record, err := h.service.RegisterFile(r.Context(), service.RegisterFileInput{
		OriginalName: req.OriginalName,
		MimeType:     req.MimeType,
		SizeBytes:    req.SizeBytes,
		StoragePath:  req.StoragePath,
		Checksum:     req.Checksum,
		Metadata:     req.Metadata,
		ExpiresAt:    req.ExpiresAt,
		Reader:       reader,
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorEnvelope{Error: message})
}
