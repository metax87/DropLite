package s3

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"droplite/internal/storage"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config 包含 S3/MinIO 存储所需的配置。
type Config struct {
	Endpoint  string // 不含协议，如 "localhost:9000" 或 "s3.amazonaws.com"
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool // 是否使用 HTTPS
	PathStyle bool // 是否使用路径风格（MinIO 需要 true）
}

// Storage 实现了 storage.Storage 接口，使用 S3 兼容存储。
type Storage struct {
	client *minio.Client
	bucket string
	region string
}

// New 创建新的 S3 存储实例。
func New(ctx context.Context, cfg Config) (*Storage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	// 检查 bucket 是否存在，不存在则创建
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{
			Region: cfg.Region,
		}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &Storage{
		client: client,
		bucket: cfg.Bucket,
		region: cfg.Region,
	}, nil
}

// Write 将文件写入 S3 存储。
func (s *Storage) Write(ctx context.Context, key string, r io.Reader) (storage.Location, error) {
	if s == nil || s.client == nil {
		return storage.Location{}, fmt.Errorf("s3 storage uninitialized")
	}

	// 清理 key 路径
	cleanKey := filepath.ToSlash(filepath.Clean(key))

	// 上传文件（使用 -1 表示未知大小，让 SDK 自动处理）
	info, err := s.client.PutObject(ctx, s.bucket, cleanKey, r, -1, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return storage.Location{}, fmt.Errorf("put object: %w", err)
	}

	return storage.Location{
		Path: cleanKey,
		URL:  fmt.Sprintf("s3://%s/%s", s.bucket, info.Key),
	}, nil
}

// Read 从 S3 存储读取文件。
func (s *Storage) Read(ctx context.Context, key string) (io.ReadCloser, error) {
	if s == nil || s.client == nil {
		return nil, fmt.Errorf("s3 storage uninitialized")
	}

	cleanKey := filepath.ToSlash(filepath.Clean(key))

	obj, err := s.client.GetObject(ctx, s.bucket, cleanKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}

	// 验证对象是否存在
	_, err = obj.Stat()
	if err != nil {
		obj.Close()
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("stat object: %w", err)
	}

	return obj, nil
}

// Delete 从 S3 存储删除文件（可选实现）。
func (s *Storage) Delete(ctx context.Context, key string) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("s3 storage uninitialized")
	}

	cleanKey := filepath.ToSlash(filepath.Clean(key))

	return s.client.RemoveObject(ctx, s.bucket, cleanKey, minio.RemoveObjectOptions{})
}
