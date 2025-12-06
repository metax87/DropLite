package api

import (
	"net/http"

	"droplite/internal/config"
	dlmiddleware "droplite/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter 构建 HTTP 路由，集中注册所有对外服务的端点。
func NewRouter(cfg *config.Config, fileHandler *FileHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(dlmiddleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(dlmiddleware.RateLimit(cfg.RateLimitRequests, cfg.RateLimitWindow))
	r.Use(dlmiddleware.Metrics())

	// 健康检查不需要鉴权
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// Prometheus 指标端点
	r.Handle("/metrics", promhttp.Handler())

	if fileHandler != nil {
		if cfg.AuthEnabled {
			// 需要鉴权的路由组
			r.Group(func(r chi.Router) {
				r.Use(dlmiddleware.APIKeyAuth(cfg.APIKeys))
				fileHandler.RegisterRoutes(r)
			})
		} else {
			// 无需鉴权（开发模式）
			fileHandler.RegisterRoutes(r)
		}
	}

	return r
}
