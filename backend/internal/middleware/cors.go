package middleware

import (
	"net/http"
	"strings"
)

// CORS 生成允许指定来源访问的跨域中间件。
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := false
	allowed := map[string]struct{}{}
	for _, origin := range allowedOrigins {
		value := strings.TrimSpace(origin)
		if value == "" {
			continue
		}
		if value == "*" {
			allowAll = true
			break
		}
		allowed[value] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowedOrigin := resolveOrigin(origin, allowAll, allowed)

			if allowedOrigin != "" {
				writeCORSHeaders(w, allowedOrigin)
			}

			if r.Method == http.MethodOptions && allowedOrigin != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func resolveOrigin(origin string, allowAll bool, allowed map[string]struct{}) string {
	if origin == "" {
		return ""
	}
	if allowAll {
		return "*"
	}
	if _, ok := allowed[origin]; ok {
		return origin
	}
	return ""
}

func writeCORSHeaders(w http.ResponseWriter, origin string) {
	headers := w.Header()
	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,PATCH,OPTIONS")
	headers.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
	headers.Set("Access-Control-Max-Age", "600")

	if origin != "*" {
		headers.Add("Vary", "Origin")
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
}
