package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/golang-jwt/jwt/v5"
)

type headerTransport struct {
	T   http.RoundTripper
	Key string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("apikey", t.Key)
	if t.T == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return t.T.RoundTrip(req)
}

// validateRemotely 通过调用 Supabase API 验证 Token
func validateRemotely(ctx context.Context, token, projectURL, anonKey string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/auth/v1/user", strings.TrimRight(projectURL, "/")), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", anonKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("remote validation failed with status: %d", resp.StatusCode)
	}

	// 解析简单的 User 结构
	var user struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("failed to decode user: %v", err)
	}
	return user.ID, nil
}

// SupabaseAuth 创建 JWT 鉴权中间件。
// 支持 HMAC (本地), JWKS (远程公钥), 和 Remote User API (直接验证)。
func SupabaseAuth(projectURL, anonKey, jwtSecret string) func(http.Handler) http.Handler {
	var jwks *keyfunc.JWKS
	var err error

	if projectURL != "" && anonKey != "" {
		jwksURL := fmt.Sprintf("%s/auth/v1/jwks", strings.TrimRight(projectURL, "/"))
		client := &http.Client{
			Transport: &headerTransport{Key: anonKey},
		}

		// 初始化 JWKS，包含自动刷新
		jwks, err = keyfunc.Get(jwksURL, keyfunc.Options{
			Client:          client,
			RefreshInterval: time.Hour,
			RefreshErrorHandler: func(err error) {
				fmt.Printf("[AuthError] JWKS refresh failed: %v\n", err)
			},
		})
		if err != nil {
			fmt.Printf("[AuthWarning] JWKS init failed (%s): %v. Will fall back to remote validation for ES256.\n", jwksURL, err)
		} else {
			fmt.Println("[AuthInfo] JWKS initialized successfully.")
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeAuthError(w, http.StatusUnauthorized, "missing Authorization header")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(authHeader, prefix) {
				writeAuthError(w, http.StatusUnauthorized, "invalid Authorization format, expected: Bearer <token>")
				return
			}

			tokenString := strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
			if tokenString == "" {
				writeAuthError(w, http.StatusUnauthorized, "empty token")
				return
			}

			// 验证逻辑：
			// 1. 尝试本地 HMAC (如果 alg=HS256 且有 Secret)
			// 2. 尝试 JWKS (如果 alg=ES256/RS256 且 JWKS 可用)
			// 3. 回退到 Remote API (如果上述都失败)

			var userID string

			// 尝试解析获取 Header
			parser := jwt.NewParser()
			unverifiedToken, _, _ := parser.ParseUnverified(tokenString, jwt.MapClaims{})

			var validatedLocally bool

			if unverifiedToken != nil {
				alg, _ := unverifiedToken.Header["alg"].(string)
				// 尝试本地解析
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
						if jwtSecret != "" {
							return []byte(jwtSecret), nil
						}
					}
					// 只有当 JWKS 初始化成功时才尝试使用 keyfunc
					if jwks != nil {
						// 简单的优化：如果 alg 不是 HMAC，尝试 JWKS
						return jwks.Keyfunc(token)
					}
					return nil, fmt.Errorf("no suitable verification method")
				})

				if err == nil && token.Valid {
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						if sub, ok := claims["sub"].(string); ok && sub != "" {
							userID = sub
							validatedLocally = true
							// fmt.Printf("[AuthDebug] Local validation success via %s\n", alg)
						}
					}
				} else {
					fmt.Printf("[AuthDebug] Local validation failed (%s): %v. Trying remote...\n", alg, err)
				}
			}

			// 如果本地验证失败，尝试远程验证
			if !validatedLocally {
				if projectURL == "" || anonKey == "" {
					writeAuthError(w, http.StatusUnauthorized, "token verification failed and remote validation not configured")
					return
				}

				uid, err := validateRemotely(r.Context(), tokenString, projectURL, anonKey)
				if err != nil {
					fmt.Printf("[AuthError] Remote validation failed: %v\n", err)
					writeAuthError(w, http.StatusUnauthorized, "invalid token (remote)")
					return
				}
				userID = uid
				fmt.Printf("[AuthDebug] Remote validation success for user: %s\n", userID)
			}

			ctx := context.WithValue(r.Context(), OwnerContextKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
