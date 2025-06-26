package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("expected MaxRetries=3, got %d", cfg.MaxRetries)
	}
	if cfg.InitialBackoff != 100*time.Millisecond {
		t.Errorf("expected InitialBackoff=100ms, got %v", cfg.InitialBackoff)
	}
	if cfg.MaxBackoff != 30*time.Second {
		t.Errorf("expected MaxBackoff=30s, got %v", cfg.MaxBackoff)
	}
	if cfg.BackoffMultiplier != 2.0 {
		t.Errorf("expected BackoffMultiplier=2.0, got %f", cfg.BackoffMultiplier)
	}
	if !cfg.Jitter {
		t.Errorf("expected Jitter=true, got %v", cfg.Jitter)
	}
}

func TestConfigBuilders(t *testing.T) {
	t.Run("WithMaxRetries", func(t *testing.T) {
		cfg := DefaultConfig().WithMaxRetries(5)
		if cfg.MaxRetries != 5 {
			t.Errorf("expected MaxRetries=5, got %d", cfg.MaxRetries)
		}
	})

	t.Run("WithInitialBackoff", func(t *testing.T) {
		cfg := DefaultConfig().WithInitialBackoff(50 * time.Millisecond)
		if cfg.InitialBackoff != 50*time.Millisecond {
			t.Errorf("expected InitialBackoff=50ms, got %v", cfg.InitialBackoff)
		}
	})

	t.Run("WithMaxBackoff", func(t *testing.T) {
		cfg := DefaultConfig().WithMaxBackoff(60 * time.Second)
		if cfg.MaxBackoff != 60*time.Second {
			t.Errorf("expected MaxBackoff=60s, got %v", cfg.MaxBackoff)
		}
	})

	t.Run("WithBackoffMultiplier", func(t *testing.T) {
		cfg := DefaultConfig().WithBackoffMultiplier(3.0)
		if cfg.BackoffMultiplier != 3.0 {
			t.Errorf("expected BackoffMultiplier=3.0, got %f", cfg.BackoffMultiplier)
		}
	})

	t.Run("WithJitter", func(t *testing.T) {
		cfg := DefaultConfig().WithJitter(false)
		if cfg.Jitter {
			t.Errorf("expected Jitter=false, got %v", cfg.Jitter)
		}
	})

	t.Run("WithIsRetryable", func(t *testing.T) {
		fn := func(err error) bool { return true }
		cfg := DefaultConfig().WithIsRetryable(fn)
		if cfg.IsRetryable == nil {
			t.Error("expected IsRetryable to be set")
		}
	})

	t.Run("chained builders", func(t *testing.T) {
		cfg := DefaultConfig().
			WithMaxRetries(5).
			WithInitialBackoff(50 * time.Millisecond).
			WithMaxBackoff(60 * time.Second).
			WithJitter(false)

		if cfg.MaxRetries != 5 {
			t.Errorf("expected MaxRetries=5, got %d", cfg.MaxRetries)
		}
		if cfg.InitialBackoff != 50*time.Millisecond {
			t.Errorf("expected InitialBackoff=50ms, got %v", cfg.InitialBackoff)
		}
		if cfg.MaxBackoff != 60*time.Second {
			t.Errorf("expected MaxBackoff=60s, got %v", cfg.MaxBackoff)
		}
		if cfg.Jitter {
			t.Errorf("expected Jitter=false, got %v", cfg.Jitter)
		}
	})
}

func TestDoSuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig()
	callCount := 0

	err := Do(ctx, cfg, func(ctx context.Context) error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}
}

func TestDoSuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithMaxRetries(3).
		WithInitialBackoff(10 * time.Millisecond).
		WithJitter(false)

	callCount := 0
	err := Do(ctx, cfg, func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestDoExhaustedRetries(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithMaxRetries(2).
		WithInitialBackoff(10 * time.Millisecond).
		WithJitter(false)

	testErr := errors.New("persistent failure")
	callCount := 0

	err := Do(ctx, cfg, func(ctx context.Context) error {
		callCount++
		return testErr
	})

	if !errors.Is(err, testErr) {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
	// MaxRetries=2 means initial attempt + 2 retries = 3 total calls
	if callCount != 3 {
		t.Errorf("expected 3 calls, got %d", callCount)
	}
}

func TestDoContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := DefaultConfig().
		WithMaxRetries(3).
		WithInitialBackoff(100 * time.Millisecond).
		WithJitter(false)

	callCount := 0
	err := Do(ctx, cfg, func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			cancel()
		}
		return errors.New("failure")
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call before cancellation, got %d", callCount)
	}
}

func TestDoContextDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg := DefaultConfig().
		WithMaxRetries(10).
		WithInitialBackoff(100 * time.Millisecond).
		WithJitter(false)

	callCount := 0
	err := Do(ctx, cfg, func(ctx context.Context) error {
		callCount++
		return errors.New("failure")
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call before deadline, got %d", callCount)
	}
}

func TestDoNonRetryableError(t *testing.T) {
	ctx := context.Background()
	nonRetryableErr := errors.New("non-retryable")
	cfg := DefaultConfig().
		WithMaxRetries(3).
		WithInitialBackoff(10 * time.Millisecond).
		WithIsRetryable(func(err error) bool {
			return !errors.Is(err, nonRetryableErr)
		})

	callCount := 0
	err := Do(ctx, cfg, func(ctx context.Context) error {
		callCount++
		return nonRetryableErr
	})

	if !errors.Is(err, nonRetryableErr) {
		t.Errorf("expected error %v, got %v", nonRetryableErr, err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call for non-retryable error, got %d", callCount)
	}
}

func TestDoExponentialBackoff(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithMaxRetries(2).
		WithInitialBackoff(10 * time.Millisecond).
		WithBackoffMultiplier(2.0).
		WithJitter(false)

	callTimes := []time.Time{}
	err := Do(ctx, cfg, func(ctx context.Context) error {
		callTimes = append(callTimes, time.Now())
		return errors.New("failure")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if len(callTimes) != 3 {
		t.Errorf("expected 3 calls, got %d", len(callTimes))
	}

	// Check that backoff increases exponentially
	if len(callTimes) >= 2 {
		backoff1 := callTimes[1].Sub(callTimes[0])
		if backoff1 < 10*time.Millisecond {
			t.Errorf("expected first backoff >= 10ms, got %v", backoff1)
		}
	}

	if len(callTimes) >= 3 {
		backoff2 := callTimes[2].Sub(callTimes[1])
		backoff1 := callTimes[1].Sub(callTimes[0])
		if backoff2 < backoff1 {
			t.Errorf("expected backoff to increase, got %v then %v", backoff1, backoff2)
		}
	}
}

func TestDoMaxBackoffCap(t *testing.T) {
	ctx := context.Background()
	maxBackoff := 50 * time.Millisecond
	cfg := DefaultConfig().
		WithMaxRetries(5).
		WithInitialBackoff(100 * time.Millisecond).
		WithMaxBackoff(maxBackoff).
		WithBackoffMultiplier(2.0).
		WithJitter(false)

	callTimes := []time.Time{}
	err := Do(ctx, cfg, func(ctx context.Context) error {
		callTimes = append(callTimes, time.Now())
		return errors.New("failure")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}

	// Verify that backoffs don't exceed maxBackoff by more than a small margin (accounting for timing variance)
	for i := 1; i < len(callTimes); i++ {
		backoff := callTimes[i].Sub(callTimes[i-1])
		// Allow 25ms margin for timing variance in CI/VM environments
		if backoff > maxBackoff+25*time.Millisecond {
			t.Errorf("backoff %v exceeds maxBackoff %v", backoff, maxBackoff)
		}
	}
}

func TestDoJitter(t *testing.T) {
	ctx := context.Background()
	cfg := DefaultConfig().
		WithMaxRetries(3).
		WithInitialBackoff(50 * time.Millisecond).
		WithJitter(true)

	backoffs := []time.Duration{}
	for run := 0; run < 3; run++ {
		callTimes := []time.Time{}
		Do(ctx, cfg, func(ctx context.Context) error {
			callTimes = append(callTimes, time.Now())
			return errors.New("failure")
		})

		if len(callTimes) >= 2 {
			backoffs = append(backoffs, callTimes[1].Sub(callTimes[0]))
		}
	}

	// With jitter enabled, backoffs should vary
	if len(backoffs) >= 2 {
		if backoffs[0] == backoffs[1] {
			t.Errorf("expected backoffs to vary with jitter enabled, got %v and %v", backoffs[0], backoffs[1])
		}
	}
}

func TestSimpleRetry(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := SimpleRetry(ctx, func(ctx context.Context) error {
		callCount++
		if callCount < 2 {
			return errors.New("failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected 2 calls, got %d", callCount)
	}
}

func TestSimpleRetryWithMaxRetries(t *testing.T) {
	ctx := context.Background()
	maxRetries := 5
	callCount := 0

	err := SimpleRetryWithMaxRetries(ctx, maxRetries, func(ctx context.Context) error {
		callCount++
		return errors.New("failure")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != maxRetries+1 {
		t.Errorf("expected %d calls, got %d", maxRetries+1, callCount)
	}
}

func TestSimpleRetryWithTimeout(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	err := SimpleRetryWithTimeout(ctx, 50*time.Millisecond, func(ctx context.Context) error {
		callCount++
		return errors.New("failure")
	})

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call before timeout, got %d", callCount)
	}
}

func TestDoNegativeMaxRetries(t *testing.T) {
	ctx := context.Background()
	cfg := Config{MaxRetries: -1}

	callCount := 0
	err := Do(ctx, cfg, func(ctx context.Context) error {
		callCount++
		return errors.New("failure")
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected 1 call with negative MaxRetries, got %d", callCount)
	}
}

func TestDoZeroInitialBackoff(t *testing.T) {
	ctx := context.Background()
	cfg := Config{
		MaxRetries:     1,
		InitialBackoff: 0, // Should be set to default
	}

	start := time.Now()
	Do(ctx, cfg, func(ctx context.Context) error {
		return errors.New("failure")
	})
	elapsed := time.Since(start)

	// Should use default backoff (100ms) even though InitialBackoff is 0
	if elapsed < 50*time.Millisecond {
		t.Errorf("expected at least 50ms backoff, got %v", elapsed)
	}
}
