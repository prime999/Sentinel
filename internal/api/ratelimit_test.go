package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiterAllowDeny(t *testing.T) {
	rl := newRateLimiter()
	window := time.Minute
	for i := 0; i < 3; i++ {
		if !rl.Allow("k", 3, window) {
			t.Fatalf("attempt %d should allow", i+1)
		}
	}
	if rl.Allow("k", 3, window) {
		t.Fatal("4th attempt should deny")
	}
}

func TestRateLimiterWindowExpiry(t *testing.T) {
	rl := newRateLimiter()
	window := 30 * time.Millisecond
	if !rl.Allow("k", 1, window) {
		t.Fatal("first should allow")
	}
	if rl.Allow("k", 1, window) {
		t.Fatal("second within window should deny")
	}
	time.Sleep(40 * time.Millisecond)
	if !rl.Allow("k", 1, window) {
		t.Fatal("after window should allow")
	}
}

func TestRateLimiterCleanupRemovesStaleKeys(t *testing.T) {
	rl := newRateLimiter()
	rl.cleanupEvery = 1
	rl.maxRetention = 20 * time.Millisecond
	if !rl.Allow("stale", 5, time.Hour) {
		t.Fatal("allow")
	}
	time.Sleep(30 * time.Millisecond)
	// Trigger cleanup via another Allow
	_ = rl.Allow("fresh", 5, time.Hour)
	rl.mu.Lock()
	_, exists := rl.buckets["stale"]
	rl.mu.Unlock()
	if exists {
		t.Fatal("stale key should be cleaned up")
	}
}

func TestRecordFailureLockout(t *testing.T) {
	rl := newRateLimiter()
	window := time.Minute
	for i := 1; i <= 4; i++ {
		count, locked := rl.RecordFailure("u", 5, window)
		if locked || count != i {
			t.Fatalf("i=%d count=%d locked=%v", i, count, locked)
		}
	}
	count, locked := rl.RecordFailure("u", 5, window)
	if !locked || count != 5 {
		t.Fatalf("want locked at 5, got count=%d locked=%v", count, locked)
	}
	// Further failures must not grow unbounded
	count2, locked2 := rl.RecordFailure("u", 5, window)
	if !locked2 || count2 != 5 {
		t.Fatalf("capped count want 5, got count=%d locked=%v", count2, locked2)
	}
	rl.Clear("u")
	if rl.Count("u", window) != 0 {
		t.Fatal("clear should reset")
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.10:12345"
	if got := clientIP(r); got != "203.0.113.10" {
		t.Fatalf("got %q", got)
	}
	r.Header.Set("X-Forwarded-For", "198.51.100.1, 203.0.113.10")
	if got := clientIP(r); got != "198.51.100.1" {
		t.Fatalf("xff got %q", got)
	}
}
