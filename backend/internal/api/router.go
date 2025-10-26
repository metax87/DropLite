package api

import (
	"net/http"

	"droplite/internal/config"
	dlmiddleware "droplite/internal/middleware"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewRouter 构建 HTTP 路由，集中注册所有对外服务的端点。
func NewRouter(cfg *config.Config) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(dlmiddleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(dlmiddleware.RateLimit(cfg.RateLimitRequests, cfg.RateLimitWindow))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}
