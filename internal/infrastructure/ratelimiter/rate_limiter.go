package ratelimiter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/config"
)

// RateLimiter defines the interface for rate limiting.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Wait(ctx context.Context, key string) error
}

// TokenBucketLimiter implements RateLimiter using token bucket algorithm.
type TokenBucketLimiter struct {
	limiters             sync.Map // map[string]*rate.Limiter
	requestsPerSecond    int
	burst                int
	cleanupInterval      time.Duration
	staleThreshold       time.Duration
	lastAccessTimestamps sync.Map // map[string]time.Time
	stopCleanup          chan struct{}
	mu                   sync.RWMutex
	isRunning            bool
}

// NewTokenBucketLimiter creates a new TokenBucketLimiter.
func NewTokenBucketLimiter(cfg *config.RateLimitConfig) *TokenBucketLimiter {
	tbl := &TokenBucketLimiter{
		requestsPerSecond: cfg.RequestsPerSecond,
		burst:             cfg.BurstSize,
		cleanupInterval:   cfg.CleanupInterval,
		staleThreshold:    cfg.StaleThreshold,
		stopCleanup:       make(chan struct{}),
		isRunning:         false,
	}

	// Start cleanup goroutine
	tbl.startCleanup()

	return tbl
}

// Allow checks if an operation is allowed for the given key.
func (tbl *TokenBucketLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("key cannot be empty")
	}

	limiter := tbl.getOrCreateLimiter(key)
	allowed := limiter.Allow()

	if allowed {
		tbl.updateLastAccess(key)
	}

	return allowed, nil
}

// Wait blocks until an operation is allowed for the given key.
func (tbl *TokenBucketLimiter) Wait(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	limiter := tbl.getOrCreateLimiter(key)
	err := limiter.Wait(ctx)

	if err == nil {
		tbl.updateLastAccess(key)
	}

	return err
}

// getOrCreateLimiter retrieves or creates a rate limiter for a key.
func (tbl *TokenBucketLimiter) getOrCreateLimiter(key string) *rate.Limiter {
	// Try to load existing limiter
	if val, exists := tbl.limiters.Load(key); exists {
		limiter, ok := val.(*rate.Limiter)
		if ok {
			return limiter
		}
	}

	// Create new limiter
	newLimiter := rate.NewLimiter(
		rate.Limit(tbl.requestsPerSecond),
		tbl.burst,
	)

	// Use LoadOrStore to handle race condition
	actual, _ := tbl.limiters.LoadOrStore(key, newLimiter)
	limiter, ok := actual.(*rate.Limiter)
	if ok {
		return limiter
	}
	return newLimiter
}

// updateLastAccess updates the last access timestamp for a key.
func (tbl *TokenBucketLimiter) updateLastAccess(key string) {
	tbl.lastAccessTimestamps.Store(key, time.Now())
}

// startCleanup starts the cleanup goroutine.
func (tbl *TokenBucketLimiter) startCleanup() {
	tbl.mu.Lock()
	if tbl.isRunning {
		tbl.mu.Unlock()
		return
	}
	tbl.isRunning = true
	tbl.mu.Unlock()

	go func() {
		ticker := time.NewTicker(tbl.cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				tbl.cleanupStaleLimiters()
			case <-tbl.stopCleanup:
				return
			}
		}
	}()
}

// cleanupStaleLimiters removes limiters that haven't been accessed recently.
func (tbl *TokenBucketLimiter) cleanupStaleLimiters() {
	now := time.Now()
	keysToDelete := make([]string, 0)

	// Find stale entries
	tbl.lastAccessTimestamps.Range(func(key, value interface{}) bool {
		lastAccess, ok := value.(time.Time)
		if !ok {
			return true
		}
		if now.Sub(lastAccess) > tbl.staleThreshold {
			keyStr, ok := key.(string)
			if ok {
				keysToDelete = append(keysToDelete, keyStr)
			}
		}
		return true
	})

	// Delete stale entries
	for _, key := range keysToDelete {
		tbl.limiters.Delete(key)
		tbl.lastAccessTimestamps.Delete(key)
	}
}

// Stop stops the cleanup goroutine.
func (tbl *TokenBucketLimiter) Stop() {
	tbl.mu.Lock()
	if !tbl.isRunning {
		tbl.mu.Unlock()
		return
	}
	tbl.isRunning = false
	tbl.mu.Unlock()

	close(tbl.stopCleanup)
}

// GetStats returns statistics about the limiters.
func (tbl *TokenBucketLimiter) GetStats() map[string]interface{} {
	count := 0
	tbl.limiters.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return map[string]interface{}{
		"active_limiters":     count,
		"requests_per_second": tbl.requestsPerSecond,
		"burst_size":          tbl.burst,
		"cleanup_interval":    tbl.cleanupInterval.String(),
		"stale_threshold":     tbl.staleThreshold.String(),
	}
}
