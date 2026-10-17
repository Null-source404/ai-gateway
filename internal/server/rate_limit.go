package server

import (
	"net/http"
	"sync"
	"time"
)

type TokenBucket struct {
	mu     sync.Mutex
	limit  int
	tokens int
	last   time.Time
}

func NewTokenBucket(limit int) *TokenBucket {
	if limit <= 0 {
		limit = 60
	}
	return &TokenBucket{limit: limit, tokens: limit, last: time.Now()}
}

func (b *TokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if now.Sub(b.last) > time.Second {
		b.tokens = b.limit
		b.last = now
	}
	if b.tokens > 0 {
		b.tokens--
		return true
	}
	return false
}

func RateLimit(next http.Handler, limit int) http.Handler {
	bucket := NewTokenBucket(limit)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !bucket.Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
