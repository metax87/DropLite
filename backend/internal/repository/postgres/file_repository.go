package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"droplite/internal/repository"
)

// NewFileRepository 返回基于 *sql.DB 的 Postgres 实现。
func NewFileRepository(db *sql.DB) *FileRepository {
	return &FileRepository{db: db}
}

// FileRepository 实现 repository.FileRepository。
type FileRepository struct {
	db *sql.DB
}

var fileSelectColumns = []string{
	"id",
	"original_name",
	"mime_type",
	"size_bytes",
	"storage_path",
	"checksum",
	"status",
	"metadata",
	"created_at",
	"updated_at",
	"expires_at",
}

var fileInsertColumns = []string{
	"id",
	"original_name",
	"mime_type",
	"size_bytes",
	"storage_path",
	"checksum",
	"status",
	"metadata",
	"expires_at",
}

// Create 插入文件记录并返回数据库生成字段（如时间戳）。
func (r *FileRepository) Create(ctx context.Context, record *repository.FileRecord) (*repository.FileRecord, error) {
	if record == nil {
		return nil, fmt.Errorf("file record is nil")
	}

	metadataBytes, err := encodeMetadata(record.Metadata)
	if err != nil {
		return nil, err
	}

	placeholders := make([]string, len(fileInsertColumns))
	for i := range fileInsertColumns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(`INSERT INTO files (%s)
	VALUES (%s)
	RETURNING %s`,
		strings.Join(fileInsertColumns, ","),
		strings.Join(placeholders, ","),
		strings.Join(fileSelectColumns, ","),
	)

	var checksum sql.NullString
	if record.Checksum != nil {
		checksum = sql.NullString{String: *record.Checksum, Valid: true}
	}

	var expires sql.NullTime
	if record.ExpiresAt != nil {
		expires = sql.NullTime{Time: *record.ExpiresAt, Valid: true}
	}

	row := r.db.QueryRowContext(
		ctx,
		query,
		record.ID,
		record.OriginalName,
		record.MimeType,
		record.SizeBytes,
		record.StoragePath,
		checksum,
		record.Status,
		metadataBytes,
		expires,
	)

	return scanFileRecord(row)
}

// GetByID 通过主键查询文件记录。
func (r *FileRepository) GetByID(ctx context.Context, id string) (*repository.FileRecord, error) {
	query := fmt.Sprintf(`SELECT %s FROM files WHERE id = $1`, strings.Join(fileSelectColumns, ","))
	row := r.db.QueryRowContext(ctx, query, id)
	file, err := scanFileRecord(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	return file, nil
}

// List 支持按状态过滤并分页。
func (r *FileRepository) List(ctx context.Context, params repository.ListFilesParams) ([]repository.FileRecord, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	args := make([]any, 0, len(params.Statuses)+2)
	whereClause := ""
	if len(params.Statuses) > 0 {
		placeholders := make([]string, len(params.Statuses))
		for i, status := range params.Statuses {
			args = append(args, status)
			placeholders[i] = fmt.Sprintf("$%d", len(args))
		}
		whereClause = "WHERE status IN (" + strings.Join(placeholders, ",") + ")"
	} else {
		// 默认排除已删除的文件
		args = append(args, repository.FileStatusDeleted)
		whereClause = fmt.Sprintf("WHERE status != $%d", len(args))
	}

	args = append(args, limit)
	limitPlaceholder := fmt.Sprintf("$%d", len(args))
	tail := fmt.Sprintf("ORDER BY created_at DESC LIMIT %s", limitPlaceholder)

	if params.Offset > 0 {
		args = append(args, params.Offset)
		offsetPlaceholder := fmt.Sprintf("$%d", len(args))
		tail += fmt.Sprintf(" OFFSET %s", offsetPlaceholder)
	}

	query := fmt.Sprintf(`SELECT %s FROM files %s %s`, strings.Join(fileSelectColumns, ","), whereClause, tail)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []repository.FileRecord
	for rows.Next() {
		rec, err := scanFileRecord(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, *rec)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// UpdateStatus 更新文件状态。
func (r *FileRepository) UpdateStatus(ctx context.Context, id string, status repository.FileStatus) error {
	query := `UPDATE files SET status = $1, updated_at = $2 WHERE id = $3`
	res, err := r.db.ExecContext(ctx, query, status, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return repository.ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanFileRecord(rs rowScanner) (*repository.FileRecord, error) {
	var (
		rec       repository.FileRecord
		checksum  sql.NullString
		metadata  []byte
		expiresAt sql.NullTime
	)

	if err := rs.Scan(
		&rec.ID,
		&rec.OriginalName,
		&rec.MimeType,
		&rec.SizeBytes,
		&rec.StoragePath,
		&checksum,
		&rec.Status,
		&metadata,
		&rec.CreatedAt,
		&rec.UpdatedAt,
		&expiresAt,
	); err != nil {
		return nil, err
	}

	if checksum.Valid {
		rec.Checksum = &checksum.String
	}
	if expiresAt.Valid {
		rec.ExpiresAt = &expiresAt.Time
	}
	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &rec.Metadata); err != nil {
			return nil, err
		}
	}
	if rec.Metadata == nil {
		rec.Metadata = map[string]any{}
	}

	return &rec, nil
}

func encodeMetadata(meta map[string]any) ([]byte, error) {
	if meta == nil {
		meta = map[string]any{}
	}
	return json.Marshal(meta)
}
