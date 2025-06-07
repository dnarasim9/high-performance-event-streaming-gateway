package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the state of the circuit breaker
type State int

const (
	// StateClosed means the circuit is operating normally
	StateClosed State = iota
	// StateOpen means the circuit is failing and rejecting calls
	StateOpen
	// StateHalfOpen means the circuit is testing recovery
	StateHalfOpen
)

// String returns the string representation of the State
func (s State) String() string {
	switch s {
	case StateClosed:
		return "Closed"
	case StateOpen:
		return "Open"
	case StateHalfOpen:
		return "HalfOpen"
	default:
		return "Unknown"
	}
}

var (
	// ErrCircuitOpen is returned when the circuit breaker is open and rejecting calls
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// Config holds the configuration for a circuit breaker
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit
	FailureThreshold int
	// ResetTimeout is the duration after which a half-open circuit will attempt to recover
	ResetTimeout time.Duration
	// HalfOpenMaxCalls is the maximum number of calls to allow in half-open state
	HalfOpenMaxCalls int
}

// OnStateChangeFunc is a callback function invoked when the circuit breaker state changes
type OnStateChangeFunc func(oldState, newState State)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu                  sync.Mutex
	state               State
	consecutiveFailures int
	lastFailureTime     time.Time
	successCount        int
	onStateChange       OnStateChangeFunc
	config              Config
}

// New creates a new circuit breaker with the given configuration
func New(config Config) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 60 * time.Second
	}
	if config.HalfOpenMaxCalls <= 0 {
		config.HalfOpenMaxCalls = 1
	}

	return &CircuitBreaker{
		state:           StateClosed,
		config:          config,
		lastFailureTime: time.Now(),
	}
}

// SetOnStateChange sets the callback function to be invoked on state changes
func (cb *CircuitBreaker) SetOnStateChange(fn OnStateChangeFunc) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.onStateChange = fn
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Execute executes the given function with circuit breaker protection.
// If the circuit is open, it returns ErrCircuitOpen immediately.
// If the function succeeds, it resets the circuit.
// If the function fails, it increments the failure counter.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return cb.executeClosed(fn)
	case StateOpen:
		return cb.executeOpen(fn)
	case StateHalfOpen:
		return cb.executeHalfOpen(fn)
	default:
		return errors.New("unknown state")
	}
}

// executeClosed executes the function while in closed state
func (cb *CircuitBreaker) executeClosed(fn func() error) error {
	err := fn()
	if err != nil {
		cb.consecutiveFailures++
		cb.lastFailureTime = time.Now()

		if cb.consecutiveFailures >= cb.config.FailureThreshold {
			cb.changeState(StateOpen)
			return err
		}
		return err
	}

	// Reset on success
	cb.consecutiveFailures = 0
	return nil
}

// executeOpen returns an error if the reset timeout hasn't passed,
// otherwise transitions to half-open state and executes the function
func (cb *CircuitBreaker) executeOpen(fn func() error) error {
	if time.Since(cb.lastFailureTime) >= cb.config.ResetTimeout {
		cb.changeState(StateHalfOpen)
		cb.successCount = 0
		// Execute the function in half-open state
		return cb.executeHalfOpen(fn)
	}
	return ErrCircuitOpen
}

// executeHalfOpen attempts the function and manages state transitions
func (cb *CircuitBreaker) executeHalfOpen(fn func() error) error {
	err := fn()
	if err != nil {
		cb.changeState(StateOpen)
		cb.lastFailureTime = time.Now()
		cb.consecutiveFailures = 1
		cb.successCount = 0
		return err
	}

	cb.successCount++
	if cb.successCount >= cb.config.HalfOpenMaxCalls {
		cb.changeState(StateClosed)
		cb.consecutiveFailures = 0
		cb.successCount = 0
	}

	return nil
}

// changeState transitions to a new state and invokes the state change callback
func (cb *CircuitBreaker) changeState(newState State) {
	oldState := cb.state
	cb.state = newState

	if cb.onStateChange != nil {
		// Call the callback outside the lock to avoid deadlocks
		go cb.onStateChange(oldState, newState)
	}
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.consecutiveFailures = 0
	cb.successCount = 0
	cb.lastFailureTime = time.Now()

	if cb.onStateChange != nil && oldState != StateClosed {
		go cb.onStateChange(oldState, StateClosed)
	}
}
