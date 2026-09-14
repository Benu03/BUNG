package main

import (
	"sync"
	"time"
)

// loginLimiter is a simple in-memory brute-force guard, keyed per
// username+IP. It's fine for a single backend replica (today's setup); if
// this ever runs with multiple replicas, swap the map for something shared
// (Redis, or a Postgres-backed counter) - the check/recordFailure/
// recordSuccess call sites in auth.go stay the same either way.
type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*attemptRecord
}

type attemptRecord struct {
	count        int
	windowStart  time.Time
	blockedUntil time.Time
}

const (
	maxLoginAttempts = 5
	loginWindow      = 5 * time.Minute
	loginLockout     = 15 * time.Minute
)

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{attempts: map[string]*attemptRecord{}}
}

// blocked reports whether key is currently locked out, and for how much
// longer.
func (l *loginLimiter) blocked(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.attempts[key]
	if !ok {
		return false, 0
	}
	if remaining := time.Until(rec.blockedUntil); remaining > 0 {
		return true, remaining
	}
	return false, 0
}

func (l *loginLimiter) recordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	rec, ok := l.attempts[key]
	if !ok || now.Sub(rec.windowStart) > loginWindow {
		rec = &attemptRecord{windowStart: now}
		l.attempts[key] = rec
	}
	rec.count++
	if rec.count >= maxLoginAttempts {
		rec.blockedUntil = now.Add(loginLockout)
	}
}

func (l *loginLimiter) recordSuccess(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}
