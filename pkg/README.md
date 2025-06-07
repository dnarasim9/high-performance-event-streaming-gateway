# Utility Packages

This directory contains reusable utility packages for the Event Streaming Gateway project.

## Packages

### 1. middleware

Contains gRPC middleware implementations for cross-cutting concerns.

#### Correlation ID Middleware (`middleware/correlation.go`)

Implements distributed request tracing through correlation IDs in gRPC services.

**Features:**
- Extracts or generates unique correlation IDs for each request
- Propagates correlation IDs through gRPC metadata and context
- Supports both unary and streaming RPC types
- Uses UUID v4 for ID generation when none exists

**Usage:**

```go
import "github.com/dheemanth-hn/event-streaming-gateway/pkg/middleware"

// Get correlation ID from context
id := middleware.CorrelationIDFromContext(ctx)

// Add correlation ID to context for downstream calls
ctx = middleware.ContextWithCorrelationID(ctx, "request-123")

// Register with gRPC server
server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(middleware.CorrelationUnaryServerInterceptor()),
    grpc.ChainStreamInterceptor(middleware.CorrelationStreamServerInterceptor()),
)
```

**API:**

- `CorrelationIDFromContext(ctx context.Context) string`
  - Extracts correlation ID from context value or metadata
  - Returns empty string if no ID found

- `ContextWithCorrelationID(ctx context.Context, id string) context.Context`
  - Adds correlation ID to context value and outgoing metadata
  - Returns new context with correlation ID set

- `CorrelationUnaryServerInterceptor() grpc.UnaryServerInterceptor`
  - Unary RPC interceptor that manages correlation IDs
  - Auto-generates UUID if not present

- `CorrelationStreamServerInterceptor() grpc.StreamServerInterceptor`
  - Stream RPC interceptor that manages correlation IDs
  - Auto-generates UUID if not present

### 2. circuitbreaker

Implements the circuit breaker pattern for fault tolerance and resilience.

#### Circuit Breaker (`circuitbreaker/circuit_breaker.go`)

Provides automatic failure handling by stopping requests to failing services.

**States:**
- **Closed**: Normal operation, all requests pass through
- **Open**: Service is failing, requests are rejected immediately
- **HalfOpen**: Testing recovery, a limited number of requests are allowed

**State Transitions:**
- Closed → Open: After reaching consecutive failure threshold
- Open → HalfOpen: After reset timeout expires
- HalfOpen → Closed: All test calls succeed
- HalfOpen → Open: A test call fails

**Features:**
- Thread-safe with sync.Mutex
- Tracks consecutive failures and transition times
- Configurable thresholds and timeouts
- State change callbacks for monitoring
- Automatic state transitions

**Usage:**

```go
import "github.com/dheemanth-hn/event-streaming-gateway/pkg/circuitbreaker"

cfg := circuitbreaker.Config{
    FailureThreshold: 5,           // Open after 5 failures
    ResetTimeout:     60 * time.Second,
    HalfOpenMaxCalls: 1,           // Test 1 request in half-open
}

cb := circuitbreaker.New(cfg)

// Set state change callback
cb.SetOnStateChange(func(old, new circuitbreaker.State) {
    log.Printf("Circuit state changed from %s to %s", old, new)
})

// Execute with circuit breaker protection
err := cb.Execute(func() error {
    return callDownstreamService()
})
if err != nil {
    if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
        return fmt.Errorf("service unavailable: circuit open")
    }
    return err
}

// Check current state
state := cb.State()

// Reset circuit manually
cb.Reset()
```

**API:**

- `New(config Config) *CircuitBreaker`
  - Creates new circuit breaker with configuration

- `Execute(fn func() error) error`
  - Executes function with circuit breaker protection
  - Returns ErrCircuitOpen if circuit is open

- `State() State`
  - Returns current circuit state (Closed, Open, HalfOpen)

- `Reset()`
  - Manually resets circuit to Closed state

- `SetOnStateChange(fn OnStateChangeFunc)`
  - Sets callback for state changes

**Configuration:**

- `FailureThreshold` (int): Consecutive failures before opening (default: 5)
- `ResetTimeout` (time.Duration): Duration before attempting recovery (default: 60s)
- `HalfOpenMaxCalls` (int): Max concurrent calls in half-open state (default: 1)

### 3. retry

Implements retry logic with exponential backoff for transient failures.

#### Retry (`retry/retry.go`)

Handles transient failures by automatically retrying with exponential backoff.

**Features:**
- Exponential backoff with configurable parameters
- Optional random jitter to avoid thundering herd
- Respects context cancellation and deadlines
- Customizable retryable error filtering
- Builder pattern configuration
- Helper functions for common scenarios

**Usage:**

```go
import "github.com/dheemanth-hn/event-streaming-gateway/pkg/retry"

ctx := context.Background()

// Using default configuration (3 retries, 100ms initial backoff)
err := retry.SimpleRetry(ctx, func(ctx context.Context) error {
    return callDownstreamService(ctx)
})

// Custom configuration
cfg := retry.DefaultConfig().
    WithMaxRetries(5).
    WithInitialBackoff(50 * time.Millisecond).
    WithMaxBackoff(30 * time.Second).
    WithJitter(true)

err = retry.Do(ctx, cfg, func(ctx context.Context) error {
    return callDownstreamService(ctx)
})

// With custom retryable error logic
cfg = retry.DefaultConfig().WithIsRetryable(func(err error) bool {
    return isTemporaryError(err) // Don't retry permanent errors
})

err = retry.Do(ctx, cfg, func(ctx context.Context) error {
    return callDownstreamService(ctx)
})

// With timeout
err = retry.SimpleRetryWithTimeout(ctx, 5*time.Second, func(ctx context.Context) error {
    return callDownstreamService(ctx)
})
```

**API:**

- `Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error`
  - Executes function with retry logic
  - Respects context cancellation
  - Returns last error if all retries exhausted

- `SimpleRetry(ctx context.Context, fn func(ctx context.Context) error) error`
  - Retries with default configuration
  - MaxRetries=3, InitialBackoff=100ms

- `SimpleRetryWithMaxRetries(ctx context.Context, maxRetries int, fn func(ctx context.Context) error) error`
  - Retries with custom max retry count

- `SimpleRetryWithTimeout(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) error) error`
  - Retries until timeout expires

**Configuration Methods:**

- `DefaultConfig() Config`
  - Returns configuration with sensible defaults

- `WithMaxRetries(maxRetries int) Config`
  - Sets maximum number of retry attempts (not including initial)

- `WithInitialBackoff(backoff time.Duration) Config`
  - Sets initial backoff duration before first retry

- `WithMaxBackoff(maxBackoff time.Duration) Config`
  - Sets maximum backoff duration (caps exponential growth)

- `WithBackoffMultiplier(multiplier float64) Config`
  - Sets exponential backoff multiplier (default: 2.0)

- `WithJitter(jitter bool) Config`
  - Enables/disables random jitter in backoff

- `WithIsRetryable(fn IsRetryableFunc) Config`
  - Sets custom function to determine if error is retryable

**Configuration:**

- `MaxRetries` (int): Number of retries (default: 3)
- `InitialBackoff` (time.Duration): First retry delay (default: 100ms)
- `MaxBackoff` (time.Duration): Maximum backoff duration (default: 30s)
- `BackoffMultiplier` (float64): Exponential backoff multiplier (default: 2.0)
- `Jitter` (bool): Enable random jitter (default: true)
- `IsRetryable` (IsRetryableFunc): Function to check if error is retryable (default: nil - retry all)

**Backoff Calculation:**

- Formula: `backoff = min(InitialBackoff * (BackoffMultiplier ^ attempt), MaxBackoff)`
- If Jitter is true: `backoff += random(0, backoff/2)`

## Examples

### Combined Usage: Resilient Service Call

```go
package example

import (
    "context"
    "errors"
    "github.com/dheemanth-hn/event-streaming-gateway/pkg/circuitbreaker"
    "github.com/dheemanth-hn/event-streaming-gateway/pkg/retry"
)

func CallRemoteServiceWithResilience(ctx context.Context, cb *circuitbreaker.CircuitBreaker, serviceURL string) ([]byte, error) {
    // Execute with circuit breaker protection
    err := cb.Execute(func() error {
        // Retry with exponential backoff
        cfg := retry.DefaultConfig().WithMaxRetries(3)
        return retry.Do(ctx, cfg, func(ctx context.Context) error {
            response, err := http.Get(serviceURL)
            if err != nil {
                return err // Retryable
            }
            defer response.Body.Close()

            if response.StatusCode >= 500 {
                return errors.New("server error") // Retryable
            }

            if response.StatusCode >= 400 {
                return errors.New("client error") // Should not retry
            }

            return nil
        })
    })

    if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
        return nil, fmt.Errorf("service unavailable: circuit open")
    }

    return body, err
}
```

## Testing

Each package includes comprehensive unit tests:

```bash
go test ./pkg/middleware/...
go test ./pkg/circuitbreaker/...
go test ./pkg/retry/...

# Run all tests with coverage
go test ./pkg/... -cover
```

## Thread Safety

- **Correlation ID Middleware**: Safe for concurrent use
- **Circuit Breaker**: Thread-safe with internal sync.Mutex
- **Retry**: Thread-safe for concurrent invocations
