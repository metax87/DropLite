package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"droplite/internal/api"
	"droplite/internal/config"
	"droplite/internal/database"
	"droplite/internal/logging"
	"droplite/internal/migrations"
	postgresrepo "droplite/internal/repository/postgres"
	"droplite/internal/service"
	"droplite/internal/storage"
	"droplite/internal/storage/local"
	s3storage "droplite/internal/storage/s3"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger := logging.New()
	logger.Println("配置加载完成，开始启动服务")

	dbCtx := context.Background()
	db, err := database.Connect(dbCtx, cfg)
	if err != nil {
		logger.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	if err := migrations.Apply(dbCtx, db); err != nil {
		logger.Fatalf("数据库迁移失败: %v", err)
	}

	fileRepo := postgresrepo.NewFileRepository(db)

	// 根据配置选择存储后端
	var fileStorage storage.Storage
	switch cfg.StorageDriver {
	case "s3":
		logger.Printf("使用 S3 存储: endpoint=%s, bucket=%s", cfg.S3Endpoint, cfg.S3Bucket)
		s3Store, err := s3storage.New(dbCtx, s3storage.Config{
			Endpoint:  cfg.S3Endpoint,
			AccessKey: cfg.S3AccessKey,
			SecretKey: cfg.S3SecretKey,
			Bucket:    cfg.S3Bucket,
			Region:    cfg.S3Region,
			UseSSL:    cfg.S3UseSSL,
			PathStyle: cfg.S3PathStyle,
		})
		if err != nil {
			logger.Fatalf("初始化 S3 存储失败: %v", err)
		}
		fileStorage = s3Store
	default:
		logger.Printf("使用本地存储: dir=%s", cfg.StorageDir)
		fileStorage = local.NewWriter(cfg.StorageDir, "")
	}

	fileService := service.NewFileService(fileRepo, fileStorage)
	fileHandler := api.NewFileHandler(fileService, cfg.MaxUploadSize)

	router := api.NewRouter(cfg, fileHandler)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      router,
	}

	logger.Printf("服务监听端口 :%s\n", cfg.HTTPPort)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("监听失败: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("优雅关闭失败: %v", err)
	}

	logger.Println("服务已停止")
}
