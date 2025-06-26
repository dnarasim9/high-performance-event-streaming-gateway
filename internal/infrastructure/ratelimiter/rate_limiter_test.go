package ratelimiter

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/config"
)

func TestTokenBucketLimiter_Allow_WithinLimit(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	// Should allow within burst size
	for i := 0; i < 5; i++ {
		allowed, err := limiter.Allow(ctx, "test-key")
		if err != nil {
			t.Errorf("Allow() error = %v, want nil", err)
		}
		if !allowed {
			t.Errorf("Allow() iteration %d returned false, want true", i)
		}
	}
}

func TestTokenBucketLimiter_Allow_ExceedsLimit(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         2,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	// Use up the burst
	limiter.Allow(ctx, "test-key")
	limiter.Allow(ctx, "test-key")

	// Next request should fail
	allowed, err := limiter.Allow(ctx, "test-key")
	if err != nil {
		t.Errorf("Allow() error = %v, want nil", err)
	}
	if allowed {
		t.Error("Allow() returned true, want false when burst exceeded")
	}
}

func TestTokenBucketLimiter_Allow_EmptyKey(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	_, err := limiter.Allow(ctx, "")
	if err == nil {
		t.Error("Allow() with empty key expected error, got nil")
	}
}

func TestTokenBucketLimiter_Wait_Success(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         10,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	// Wait should succeed
	err := limiter.Wait(ctx, "test-key")
	if err != nil {
		t.Errorf("Wait() error = %v, want nil", err)
	}
}

func TestTokenBucketLimiter_Wait_ContextCancellation(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 1,
		BurstSize:         1,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	// Use up the burst
	limiter.Allow(ctx, "test-key")

	// Cancel context
	cancel()

	// Wait with cancelled context should fail
	err := limiter.Wait(ctx, "test-key")
	if err == nil {
		t.Error("Wait() with cancelled context expected error, got nil")
	}
}

func TestTokenBucketLimiter_ConcurrentAccess(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         50,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	var wg sync.WaitGroup
	allowedCount := 0
	var mu sync.Mutex

	// Spawn multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				allowed, _ := limiter.Allow(ctx, "concurrent-key")
				if allowed {
					mu.Lock()
					allowedCount++
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	// Should allow some but not all due to burst limit
	if allowedCount <= 0 {
		t.Error("ConcurrentAccess() should allow some requests")
	}
	if allowedCount > 50 {
		t.Errorf("ConcurrentAccess() allowed %d requests, expected <= 50", allowedCount)
	}
}

func TestTokenBucketLimiter_MultipleKeys(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         3,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	// Each key should have its own limiter
	allowed1, _ := limiter.Allow(ctx, "key-1")
	allowed2, _ := limiter.Allow(ctx, "key-2")

	if !allowed1 || !allowed2 {
		t.Error("Allow() on different keys should both succeed")
	}

	// Get stats to verify multiple limiters exist
	stats := limiter.GetStats()
	if activeCount, ok := stats["active_limiters"].(int); ok {
		if activeCount < 2 {
			t.Errorf("GetStats() active_limiters = %d, want >= 2", activeCount)
		}
	}
}

func TestTokenBucketLimiter_RateRecovery(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 1000,
		BurstSize:         1,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	// Use the burst
	allowed1, _ := limiter.Allow(ctx, "test-key")
	if !allowed1 {
		t.Error("First Allow() should succeed")
	}

	// Next should fail
	allowed2, _ := limiter.Allow(ctx, "test-key")
	if allowed2 {
		t.Error("Second Allow() should fail due to burst exhaustion")
	}

	// Wait for tokens to recover
	time.Sleep(2 * time.Millisecond)

	// Should be allowed again
	allowed3, _ := limiter.Allow(ctx, "test-key")
	if !allowed3 {
		t.Error("Allow() after recovery should succeed")
	}
}

func TestTokenBucketLimiter_Stats(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 50,
		BurstSize:         10,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()
	limiter.Allow(ctx, "test-key")

	stats := limiter.GetStats()

	if rps, ok := stats["requests_per_second"].(int); ok {
		if rps != 50 {
			t.Errorf("GetStats() requests_per_second = %d, want 50", rps)
		}
	}

	if burst, ok := stats["burst_size"].(int); ok {
		if burst != 10 {
			t.Errorf("GetStats() burst_size = %d, want 10", burst)
		}
	}
}

func TestTokenBucketLimiter_Stop(t *testing.T) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 10,
		BurstSize:         5,
		CleanupInterval:   100 * time.Millisecond,
		StaleThreshold:    1 * time.Second,
	}

	limiter := NewTokenBucketLimiter(cfg)

	ctx := context.Background()
	limiter.Allow(ctx, "test-key")

	// Stop should cleanly shutdown
	limiter.Stop()

	// Verify cleanup goroutine is stopped
	time.Sleep(200 * time.Millisecond)

	// Stop is idempotent
	limiter.Stop()
}
