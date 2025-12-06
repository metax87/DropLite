package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 聚合服务启动需要的关键配置。
type Config struct {
	HTTPPort           string
	StorageDir         string
	CORSAllowedOrigins []string
	RateLimitRequests  int
	RateLimitWindow    time.Duration
	DBHost             string
	DBPort             int
	DBUser             string
	DBPassword         string
	DBName             string
	DBSSLMode          string
	// 鉴权配置
	AuthEnabled bool     // 是否启用 API Key 鉴权
	APIKeys     []string // 有效的 API Keys 列表
	// 存储配置
	StorageDriver string // "local" 或 "s3"
	S3Endpoint    string // S3/MinIO 端点，不含协议
	S3AccessKey   string
	S3SecretKey   string
	S3Bucket      string
	S3Region      string
	S3UseSSL      bool // 是否使用 HTTPS
	S3PathStyle   bool // 是否使用路径风格访问（MinIO 需要设为 true）
}

// Load 从环境变量加载配置，并提供默认值。
func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	storage := os.Getenv("STORAGE_DIR")
	if storage == "" {
		storage = "./data"
	}

	if err := ensureDir(storage); err != nil {
		return nil, fmt.Errorf("确保存储目录失败: %w", err)
	}

	corsOrigins := parseList(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if len(corsOrigins) == 0 {
		corsOrigins = []string{"http://localhost:5173"}
	}

	rateLimitRequests, err := parseIntEnv("RATE_LIMIT_REQUESTS", 60)
	if err != nil {
		return nil, err
	}

	rateLimitWindow, err := parseDurationEnv("RATE_LIMIT_WINDOW", time.Minute)
	if err != nil {
		return nil, err
	}

	dbPort, err := parseIntEnv("DB_PORT", 5432)
	if err != nil {
		return nil, err
	}

	// 鉴权配置
	authEnabled := parseBoolEnv("AUTH_ENABLED", true)
	apiKeys := parseList(os.Getenv("API_KEYS"))
	if len(apiKeys) == 0 {
		// 开发环境默认 key
		apiKeys = []string{"dev-api-key-123456"}
	}

	// 存储配置
	storageDriver := envOrDefault("STORAGE_DRIVER", "local")

	return &Config{
		HTTPPort:           port,
		StorageDir:         storage,
		CORSAllowedOrigins: corsOrigins,
		RateLimitRequests:  rateLimitRequests,
		RateLimitWindow:    rateLimitWindow,
		DBHost:             envOrDefault("DB_HOST", "127.0.0.1"),
		DBPort:             dbPort,
		DBUser:             envOrDefault("DB_USER", "droplite"),
		DBPassword:         envOrDefault("DB_PASSWORD", "droplite"),
		DBName:             envOrDefault("DB_NAME", "droplite"),
		DBSSLMode:          envOrDefault("DB_SSL_MODE", "disable"),
		AuthEnabled:        authEnabled,
		APIKeys:            apiKeys,
		StorageDriver:      storageDriver,
		S3Endpoint:         envOrDefault("S3_ENDPOINT", "localhost:9000"),
		S3AccessKey:        envOrDefault("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:        envOrDefault("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:           envOrDefault("S3_BUCKET", "droplite"),
		S3Region:           envOrDefault("S3_REGION", "us-east-1"),
		S3UseSSL:           parseBoolEnv("S3_USE_SSL", false),
		S3PathStyle:        parseBoolEnv("S3_PATH_STYLE", true),
	}, nil
}

func ensureDir(path string) error {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("路径 %s 已存在但不是目录", path)
		}
		return nil
	}

	if os.IsNotExist(err) {
		return os.MkdirAll(path, 0o755)
	}

	return err
}

func parseList(raw string) []string {
	if raw == "" {
		return nil
	}

	items := strings.Split(raw, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func parseIntEnv(key string, defaultValue int) (int, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("解析 %s 失败: %w", key, err)
	}
	if value <= 0 {
		return defaultValue, nil
	}
	return value, nil
}

func parseDurationEnv(key string, defaultValue time.Duration) (time.Duration, error) {
	raw := os.Getenv(key)
	if raw == "" {
		return defaultValue, nil
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("解析 %s 失败: %w", key, err)
	}
	if value <= 0 {
		return defaultValue, nil
	}
	return value, nil
}

func parseBoolEnv(key string, defaultValue bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return defaultValue
	}
	lower := strings.ToLower(raw)
	return lower == "true" || lower == "1" || lower == "yes"
}

// PostgresDSN 生成标准 postgres:// 连接串，供数据访问层直接使用。
func (c *Config) PostgresDSN() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   fmt.Sprintf("%s:%d", c.DBHost, c.DBPort),
		Path:   c.DBName,
	}

	q := url.Values{}
	if c.DBSSLMode != "" {
		q.Set("sslmode", c.DBSSLMode)
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
