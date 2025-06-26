package circuitbreaker

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	t.Run("creates circuit breaker with custom config", func(t *testing.T) {
		cfg := Config{
			FailureThreshold: 3,
			ResetTimeout:     10 * time.Second,
			HalfOpenMaxCalls: 2,
		}

		cb := New(cfg)

		if cb.State() != StateClosed {
			t.Errorf("expected initial state to be Closed, got %v", cb.State())
		}
	})

	t.Run("sets default values for missing config", func(t *testing.T) {
		cfg := Config{}
		cb := New(cfg)

		if cb.config.FailureThreshold != 5 {
			t.Errorf("expected default FailureThreshold to be 5, got %d", cb.config.FailureThreshold)
		}
		if cb.config.ResetTimeout != 60*time.Second {
			t.Errorf("expected default ResetTimeout to be 60s, got %v", cb.config.ResetTimeout)
		}
		if cb.config.HalfOpenMaxCalls != 1 {
			t.Errorf("expected default HalfOpenMaxCalls to be 1, got %d", cb.config.HalfOpenMaxCalls)
		}
	})
}

func TestCircuitBreakerClosed(t *testing.T) {
	t.Run("executes function successfully", func(t *testing.T) {
		cb := New(Config{FailureThreshold: 3})

		callCount := 0
		fn := func() error {
			callCount++
			return nil
		}

		err := cb.Execute(fn)

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if callCount != 1 {
			t.Errorf("expected function to be called once, was called %d times", callCount)
		}
		if cb.State() != StateClosed {
			t.Errorf("expected state to remain Closed, got %v", cb.State())
		}
	})

	t.Run("resets failure counter on success", func(t *testing.T) {
		cb := New(Config{FailureThreshold: 3})

		// First failure
		cb.Execute(func() error { return errors.New("fail") })
		cb.Execute(func() error { return errors.New("fail") })

		// Success
		err := cb.Execute(func() error { return nil })
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}

		// Should still be in Closed state after success
		if cb.State() != StateClosed {
			t.Errorf("expected state to be Closed, got %v", cb.State())
		}

		// One more failure should be fine (counter was reset)
		cb.Execute(func() error { return errors.New("fail") })
		if cb.State() != StateClosed {
			t.Errorf("expected state to remain Closed after one failure post-reset, got %v", cb.State())
		}
	})

	t.Run("opens circuit after reaching failure threshold", func(t *testing.T) {
		cb := New(Config{FailureThreshold: 2})

		// First failure
		cb.Execute(func() error { return errors.New("fail") })
		if cb.State() != StateClosed {
			t.Errorf("expected state to be Closed after 1 failure, got %v", cb.State())
		}

		// Second failure - should open
		cb.Execute(func() error { return errors.New("fail") })
		if cb.State() != StateOpen {
			t.Errorf("expected state to be Open after 2 failures, got %v", cb.State())
		}
	})

	t.Run("returns error to caller when function fails", func(t *testing.T) {
		cb := New(Config{FailureThreshold: 3})

		testErr := errors.New("test error")
		err := cb.Execute(func() error { return testErr })

		if !errors.Is(err, testErr) {
			t.Errorf("expected error %v, got %v", testErr, err)
		}
	})
}

func TestCircuitBreakerOpen(t *testing.T) {
	t.Run("rejects calls with ErrCircuitOpen", func(t *testing.T) {
		cb := New(Config{
			FailureThreshold: 1,
			ResetTimeout:     100 * time.Millisecond,
		})

		// Open the circuit
		cb.Execute(func() error { return errors.New("fail") })

		// Subsequent call should be rejected immediately
		callCount := 0
		err := cb.Execute(func() error {
			callCount++
			return nil
		})

		if !errors.Is(err, ErrCircuitOpen) {
			t.Errorf("expected ErrCircuitOpen, got %v", err)
		}
		if callCount != 0 {
			t.Errorf("expected function not to be called, was called %d times", callCount)
		}
	})

	t.Run("transitions to half-open after timeout", func(t *testing.T) {
		resetTimeout := 50 * time.Millisecond
		cb := New(Config{
			FailureThreshold: 1,
			ResetTimeout:     resetTimeout,
		})

		// Open the circuit
		cb.Execute(func() error { return errors.New("fail") })

		// Wait for reset timeout
		time.Sleep(resetTimeout + 10*time.Millisecond)

		// Next call should transition to half-open and attempt execution
		err := cb.Execute(func() error { return nil })

		if err != nil {
			t.Errorf("expected no error in half-open, got %v", err)
		}
		if cb.State() != StateClosed {
			t.Errorf("expected state to be Closed after successful half-open attempt, got %v", cb.State())
		}
	})
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	t.Run("closes circuit on successful call", func(t *testing.T) {
		resetTimeout := 50 * time.Millisecond
		cb := New(Config{
			FailureThreshold: 1,
			ResetTimeout:     resetTimeout,
			HalfOpenMaxCalls: 1,
		})

		// Open the circuit
		cb.Execute(func() error { return errors.New("fail") })

		// Wait for reset timeout
		time.Sleep(resetTimeout + 10*time.Millisecond)

		// Successful call in half-open should close the circuit
		err := cb.Execute(func() error { return nil })

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if cb.State() != StateClosed {
			t.Errorf("expected state to be Closed, got %v", cb.State())
		}
	})

	t.Run("reopens circuit on failed call", func(t *testing.T) {
		resetTimeout := 50 * time.Millisecond
		cb := New(Config{
			FailureThreshold: 1,
			ResetTimeout:     resetTimeout,
			HalfOpenMaxCalls: 1,
		})

		// Open the circuit
		cb.Execute(func() error { return errors.New("fail") })

		// Wait for reset timeout
		time.Sleep(resetTimeout + 10*time.Millisecond)

		// Failed call in half-open should reopen the circuit
		testErr := errors.New("test error")
		err := cb.Execute(func() error { return testErr })

		if !errors.Is(err, testErr) {
			t.Errorf("expected error %v, got %v", testErr, err)
		}
		if cb.State() != StateOpen {
			t.Errorf("expected state to be Open, got %v", cb.State())
		}
	})

	t.Run("allows multiple calls in half-open state", func(t *testing.T) {
		resetTimeout := 50 * time.Millisecond
		cb := New(Config{
			FailureThreshold: 1,
			ResetTimeout:     resetTimeout,
			HalfOpenMaxCalls: 3,
		})

		// Open the circuit
		cb.Execute(func() error { return errors.New("fail") })

		// Wait for reset timeout
		time.Sleep(resetTimeout + 10*time.Millisecond)

		// Multiple successful calls
		callCount := 0
		for i := 0; i < 3; i++ {
			err := cb.Execute(func() error {
				callCount++
				return nil
			})
			if err != nil {
				t.Errorf("call %d failed: %v", i+1, err)
			}
		}

		if callCount != 3 {
			t.Errorf("expected 3 calls in half-open, got %d", callCount)
		}
		if cb.State() != StateClosed {
			t.Errorf("expected state to be Closed after all half-open calls succeeded, got %v", cb.State())
		}
	})
}

func TestCircuitBreakerReset(t *testing.T) {
	t.Run("resets state to closed", func(t *testing.T) {
		cb := New(Config{FailureThreshold: 1})

		// Open the circuit
		cb.Execute(func() error { return errors.New("fail") })

		// Reset
		cb.Reset()

		if cb.State() != StateClosed {
			t.Errorf("expected state to be Closed after reset, got %v", cb.State())
		}

		// Should accept calls again
		err := cb.Execute(func() error { return nil })
		if err != nil {
			t.Errorf("expected no error after reset, got %v", err)
		}
	})
}

func TestCircuitBreakerStateChange(t *testing.T) {
	t.Run("invokes state change callback", func(t *testing.T) {
		cb := New(Config{FailureThreshold: 1})

		var stateChanges []struct {
			old, new State
		}
		mu := sync.Mutex{}

		cb.SetOnStateChange(func(old, new State) {
			mu.Lock()
			defer mu.Unlock()
			stateChanges = append(stateChanges, struct{ old, new State }{old, new})
		})

		// Open the circuit
		cb.Execute(func() error { return errors.New("fail") })

		// Give callback time to execute
		time.Sleep(10 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()

		if len(stateChanges) == 0 {
			t.Error("expected state change callback to be invoked")
		}
		if len(stateChanges) > 0 {
			if stateChanges[0].old != StateClosed || stateChanges[0].new != StateOpen {
				t.Errorf("expected state change from Closed to Open, got from %v to %v", stateChanges[0].old, stateChanges[0].new)
			}
		}
	})
}

func TestCircuitBreakerConcurrency(t *testing.T) {
	t.Run("handles concurrent calls safely", func(t *testing.T) {
		cb := New(Config{FailureThreshold: 100})

		var wg sync.WaitGroup
		var mu sync.Mutex
		var results []error

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(attempt int) {
				defer wg.Done()

				var err error
				if attempt%10 == 0 {
					err = cb.Execute(func() error { return errors.New("fail") })
				} else {
					err = cb.Execute(func() error { return nil })
				}

				mu.Lock()
				results = append(results, err)
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		if len(results) != 100 {
			t.Errorf("expected 100 results, got %d", len(results))
		}

		// Should still be in Closed state (didn't reach threshold)
		if cb.State() != StateClosed {
			t.Errorf("expected state to be Closed, got %v", cb.State())
		}
	})
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateClosed, "Closed"},
		{StateOpen, "Open"},
		{StateHalfOpen, "HalfOpen"},
		{State(999), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.expected)
		}
	}
}
