package middleware

import (
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
}

type visitor struct {
	lastSeen time.Time
	count    int
}

var limiter = &rateLimiter{
	visitors: make(map[string]*visitor),
}

func RateLimitMiddleware() func(http.Handler) http.Handler {
	// Clean up old entries periodically
	go func() {
		for {
			time.Sleep(time.Minute)
			limiter.mu.Lock()
			for ip, v := range limiter.visitors {
				if time.Since(v.lastSeen) > time.Minute {
					delete(limiter.visitors, ip)
				}
			}
			limiter.mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr

			limiter.mu.Lock()
			v, exists := limiter.visitors[ip]
			if !exists {
				limiter.visitors[ip] = &visitor{lastSeen: time.Now(), count: 1}
				limiter.mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			// Reset count if window expired
			if time.Since(v.lastSeen) > time.Minute {
				v.count = 1
				v.lastSeen = time.Now()
			} else {
				v.count++
			}

			// Check rate limit (100 requests per minute)
			if v.count > 100 {
				limiter.mu.Unlock()
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			limiter.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}
