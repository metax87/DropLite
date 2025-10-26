package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimit 限制同一来源在指定窗口内的请求数量。
func RateLimit(maxRequests int, window time.Duration) func(http.Handler) http.Handler {
	if maxRequests <= 0 || window <= 0 {
		return passthrough
	}

	limiter := &ipRateLimiter{
		maxRequests: maxRequests,
		window:      window,
		clients:     make(map[string]*clientCounter),
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(clientKey(r)) {
				retryAfter := strconv.Itoa(int(window.Seconds()))
				w.Header().Set("Retry-After", retryAfter)
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func passthrough(next http.Handler) http.Handler {
	return next
}

type ipRateLimiter struct {
	mu          sync.Mutex
	clients     map[string]*clientCounter
	maxRequests int
	window      time.Duration
}

type clientCounter struct {
	count   int
	expires time.Time
}

func (l *ipRateLimiter) Allow(key string) bool {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.clients[key]
	if !ok || now.After(entry.expires) {
		l.clients[key] = &clientCounter{
			count:   1,
			expires: now.Add(l.window),
		}
		return true
	}

	if entry.count >= l.maxRequests {
		return false
	}

	entry.count++

	if len(l.clients) > 1024 {
		l.cleanupLocked(now)
	}

	return true
}

func (l *ipRateLimiter) cleanupLocked(now time.Time) {
	for key, entry := range l.clients {
		if now.After(entry.expires) {
			delete(l.clients, key)
		}
	}
}

func clientKey(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
