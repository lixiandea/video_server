package middleware

import (
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// 限流器按客户端 IP 无上限增长会造成内存泄漏（P2-3），
// 这里直接操纵 lastSeen 来验证回收逻辑，避免测试真的等待 TTL。
func TestRateLimiterCleanupEvictsOnlyIdleEntries(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(10), 20)
	rl.getLimiter("10.0.0.1")
	rl.getLimiter("10.0.0.2")

	if got := rl.Size(); got != 2 {
		t.Fatalf("expected 2 tracked limiters, got %d", got)
	}

	rl.mu.RLock()
	idle := rl.limiters["10.0.0.1"]
	rl.mu.RUnlock()
	idle.lastSeen.Store(time.Now().Add(-2 * time.Hour).Unix())

	if removed := rl.Cleanup(time.Hour); removed != 1 {
		t.Fatalf("expected 1 evicted limiter, got %d", removed)
	}
	if got := rl.Size(); got != 1 {
		t.Fatalf("expected 1 remaining limiter, got %d", got)
	}

	rl.mu.RLock()
	_, stillTracked := rl.limiters["10.0.0.2"]
	rl.mu.RUnlock()
	if !stillTracked {
		t.Fatal("active limiter should not be evicted")
	}
}

// 回收后再访问同一 IP 应重新建立限流器，且新建的桶是全额的。
func TestRateLimiterRecreatesEvictedEntry(t *testing.T) {
	rl := NewRateLimiter(rate.Limit(1), 1)

	first := rl.getLimiter("10.0.0.3")
	if !first.Allow() {
		t.Fatal("first request should be allowed")
	}
	if first.Allow() {
		t.Fatal("second request should be rate limited")
	}

	rl.mu.Lock()
	delete(rl.limiters, "10.0.0.3")
	rl.mu.Unlock()

	second := rl.getLimiter("10.0.0.3")
	if !second.Allow() {
		t.Fatal("recreated limiter should start with a full bucket")
	}
}
