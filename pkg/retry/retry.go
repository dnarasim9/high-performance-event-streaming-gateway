package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// IsRetryableFunc is a function type that determines if an error is retryable
type IsRetryableFunc func(error) bool

// Config holds the configuration for retry behavior
type Config struct {
	// MaxRetries is the maximum number of retry attempts (not including the initial attempt)
	MaxRetries int
	// InitialBackoff is the duration to wait before the first retry
	InitialBackoff time.Duration
	// MaxBackoff is the maximum duration to wait between retries
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
	// Jitter enables random jitter to be added to backoff durations
	Jitter bool
	// IsRetryable is a function to determine if an error is retryable
	// If nil, all errors are considered retryable
	IsRetryable IsRetryableFunc
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
		Jitter:            true,
		IsRetryable:       nil,
	}
}

// WithMaxRetries returns a new config with the specified max retries
func (c Config) WithMaxRetries(maxRetries int) Config {
	c.MaxRetries = maxRetries
	return c
}

// WithInitialBackoff returns a new config with the specified initial backoff
func (c Config) WithInitialBackoff(backoff time.Duration) Config {
	c.InitialBackoff = backoff
	return c
}

// WithMaxBackoff returns a new config with the specified max backoff
func (c Config) WithMaxBackoff(maxBackoff time.Duration) Config {
	c.MaxBackoff = maxBackoff
	return c
}

// WithBackoffMultiplier returns a new config with the specified backoff multiplier
func (c Config) WithBackoffMultiplier(multiplier float64) Config {
	c.BackoffMultiplier = multiplier
	return c
}

// WithJitter returns a new config with jitter enabled/disabled
func (c Config) WithJitter(jitter bool) Config {
	c.Jitter = jitter
	return c
}

// WithIsRetryable returns a new config with the specified retryable function
func (c Config) WithIsRetryable(isRetryable IsRetryableFunc) Config {
	c.IsRetryable = isRetryable
	return c
}

// Do executes the given function with retry logic, respecting exponential backoff
// and context cancellation.
// It returns the error from the function if all retries are exhausted,
// or the context error if the context is canceled.
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error {
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 100 * time.Millisecond
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = 30 * time.Second
	}
	if cfg.BackoffMultiplier <= 0 {
		cfg.BackoffMultiplier = 2.0
	}

	var lastErr error

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		// Check if context is already canceled before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		// Check if we should retry this error
		if cfg.IsRetryable != nil && !cfg.IsRetryable(lastErr) {
			return lastErr
		}

		// Don't wait after the last attempt
		if attempt < cfg.MaxRetries {
			backoff := calculateBackoff(attempt, cfg)

			// Wait with context cancellation awareness
			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return lastErr
}

// calculateBackoff calculates the backoff duration for the given attempt
func calculateBackoff(attempt int, cfg Config) time.Duration {
	// Exponential backoff: initialBackoff * (multiplier ^ attempt)
	backoffDuration := time.Duration(float64(cfg.InitialBackoff) * math.Pow(cfg.BackoffMultiplier, float64(attempt)))

	// Cap at MaxBackoff
	if backoffDuration > cfg.MaxBackoff {
		backoffDuration = cfg.MaxBackoff
	}

	// Add jitter if enabled
	if cfg.Jitter {
		//nolint:gosec // jitter doesn't need cryptographic randomness
		jitterAmount := time.Duration(rand.Int63n(int64(backoffDuration) / 2))
		backoffDuration += jitterAmount
	}

	return backoffDuration
}

// SimpleRetry is a helper function that retries a function with default configuration
// MaxRetries defaults to 3, with exponential backoff starting at 100ms
func SimpleRetry(ctx context.Context, fn func(ctx context.Context) error) error {
	return Do(ctx, DefaultConfig(), fn)
}

// SimpleRetryWithMaxRetries is a helper function that retries a function with a custom max retry count
func SimpleRetryWithMaxRetries(ctx context.Context, maxRetries int, fn func(ctx context.Context) error) error {
	cfg := DefaultConfig().WithMaxRetries(maxRetries)
	return Do(ctx, cfg, fn)
}

// SimpleRetryWithTimeout is a helper function that retries a function until the given timeout
func SimpleRetryWithTimeout(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg := DefaultConfig()
	return Do(ctx, cfg, fn)
}
