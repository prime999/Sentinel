package api

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const rateLimitCleanupEvery = 64

// rateLimiter is an in-memory sliding-window limiter.
type rateLimiter struct {
	mu            sync.Mutex
	buckets       map[string][]time.Time
	calls         int
	cleanupEvery  int
	maxRetention  time.Duration // prune timestamps older than this during cleanup
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		buckets:      make(map[string][]time.Time),
		cleanupEvery: rateLimitCleanupEvery,
		maxRetention: 2 * time.Hour,
	}
}

// Allow records an event for key and returns true if under limit within window.
func (rl *rateLimiter) Allow(key string, limit int, window time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.maybeCleanupLocked(now)

	times := pruneTimes(rl.buckets[key], cutoff)
	if len(times) >= limit {
		rl.buckets[key] = times
		return false
	}
	rl.buckets[key] = append(times, now)
	return true
}

// Count returns the number of events for key still inside window (does not record).
func (rl *rateLimiter) Count(key string, window time.Duration) int {
	now := time.Now()
	cutoff := now.Add(-window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	times := pruneTimes(rl.buckets[key], cutoff)
	if len(times) == 0 {
		delete(rl.buckets, key)
	} else {
		rl.buckets[key] = times
	}
	return len(times)
}

// Clear removes all events for key.
func (rl *rateLimiter) Clear(key string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.buckets, key)
}

// RecordFailure appends a failure unless the key is already at limit (caps growth while locked).
// Returns the failure count after the call and whether the account is locked (count >= limit).
func (rl *rateLimiter) RecordFailure(key string, limit int, window time.Duration) (count int, locked bool) {
	now := time.Now()
	cutoff := now.Add(-window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.maybeCleanupLocked(now)

	times := pruneTimes(rl.buckets[key], cutoff)
	if len(times) >= limit {
		rl.buckets[key] = times
		return len(times), true
	}
	times = append(times, now)
	rl.buckets[key] = times
	return len(times), len(times) >= limit
}

func (rl *rateLimiter) maybeCleanupLocked(now time.Time) {
	rl.calls++
	if rl.cleanupEvery <= 0 || rl.calls < rl.cleanupEvery {
		return
	}
	rl.calls = 0
	cutoff := now.Add(-rl.maxRetention)
	for key, times := range rl.buckets {
		pruned := pruneTimes(times, cutoff)
		if len(pruned) == 0 {
			delete(rl.buckets, key)
		} else {
			rl.buckets[key] = pruned
		}
	}
}

func pruneTimes(times []time.Time, cutoff time.Time) []time.Time {
	if len(times) == 0 {
		return nil
	}
	n := 0
	for _, t := range times {
		if t.After(cutoff) {
			times[n] = t
			n++
		}
	}
	return times[:n]
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if ip := strings.TrimSpace(parts[0]); ip != "" {
			return ip
		}
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
