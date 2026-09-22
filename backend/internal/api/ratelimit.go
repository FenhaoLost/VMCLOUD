package api

import (
	"net/http"
	"sync"
	"time"
)

// loginRateLimiter throttles failed login attempts per identity
// (client IP + username / access code) to mitigate brute-force attacks.
// A sliding window keeps only failures within the window; successful
// logins reset the counter for that identity.
type loginRateLimiter struct {
	mu      sync.Mutex
	window  time.Duration
	maxFail int
	fails   map[string][]time.Time
}

var loginLimiter = &loginRateLimiter{
	window:  10 * time.Minute,
	maxFail: 5,
	fails:   map[string][]time.Time{},
}

// allow reports whether a login attempt for key may proceed.
func (l *loginRateLimiter) allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-l.window)
	l.mu.Lock()
	defer l.mu.Unlock()
	kept := l.fails[key][:0]
	for _, t := range l.fails[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.fails[key] = kept
	return len(kept) < l.maxFail
}

// recordFail registers a failed login attempt for key.
func (l *loginRateLimiter) recordFail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.fails[key] = append(l.fails[key], time.Now())
}

// reset clears all failed attempts for key after a successful login.
func (l *loginRateLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fails, key)
}

// loginRateLimited is a tiny wrapper that applies the limiter and writes a
// 429 response when the identity is currently throttled.
func loginRateLimited(w http.ResponseWriter, key string) bool {
	if !loginLimiter.allow(key) {
		jsonResponse(w, http.StatusTooManyRequests, APIResponse{Success: false, Message: "登录尝试过于频繁，请稍后再试"})
		return true
	}
	return false
}
