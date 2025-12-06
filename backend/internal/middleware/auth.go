package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// OwnerContextKey 是存储在 context 中的 owner ID 的键。
type OwnerContextKey struct{}

// APIKeyAuth 创建 API Key 鉴权中间件。
// 期望请求头格式：Authorization: ApiKey <token>
// 验证成功后将 API Key 作为 owner_id 存入 context。
func APIKeyAuth(validKeys []string) func(http.Handler) http.Handler {
	keySet := make(map[string]struct{}, len(validKeys))
	for _, key := range validKeys {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			keySet[trimmed] = struct{}{}
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			// 期望格式: "ApiKey <token>"
			const prefix = "ApiKey "
			if !strings.HasPrefix(authHeader, prefix) {
				writeAuthError(w, http.StatusUnauthorized, "invalid Authorization format, expected: ApiKey <token>")
				return
			}

			apiKey := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
			if apiKey == "" {
				writeAuthError(w, http.StatusUnauthorized, "empty API key")
				return
			}

			if _, valid := keySet[apiKey]; !valid {
				writeAuthError(w, http.StatusUnauthorized, "invalid API key")
				return
			}

			// 将 API Key 作为 owner_id 存入 context
			ctx := context.WithValue(r.Context(), OwnerContextKey{}, apiKey)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetOwnerID 从 context 中获取经过鉴权的 owner ID。
func GetOwnerID(ctx context.Context) string {
	if v, ok := ctx.Value(OwnerContextKey{}).(string); ok {
		return v
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", `ApiKey realm="DropLite API"`)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
