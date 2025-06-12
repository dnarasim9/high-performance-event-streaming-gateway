package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Health status constants.
const (
	HealthyStatus   = "healthy"
	UnhealthyStatus = "unhealthy"
)

// Check defines the interface for health checks.
type Check interface {
	Name() string
	Check(ctx context.Context) error
}

// CheckStatus represents the status of a single health check.
type CheckStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// HealthStatus represents the overall health status.
type HealthStatus struct { //nolint:revive // established API
	Status    string        `json:"status"`
	Checks    []CheckStatus `json:"checks"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthChecker manages health checks for various components.
type HealthChecker struct { //nolint:revive // established API
	checks map[string]Check
	logger *slog.Logger
	mu     sync.RWMutex
}

// NewHealthChecker creates a new HealthChecker.
func NewHealthChecker(logger *slog.Logger) *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]Check),
		logger: logger,
	}
}

// RegisterCheck registers a health check.
func (hc *HealthChecker) RegisterCheck(check Check) error {
	if check == nil {
		return fmt.Errorf("check cannot be nil")
	}

	name := check.Name()
	if name == "" {
		return fmt.Errorf("check name cannot be empty")
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	if _, exists := hc.checks[name]; exists {
		return fmt.Errorf("check with name %q already registered", name)
	}

	hc.checks[name] = check
	hc.logger.Info("registered health check", "check_name", name)
	return nil
}

// CheckAll runs all registered health checks and returns the overall status.
func (hc *HealthChecker) CheckAll(ctx context.Context) *HealthStatus {
	hc.mu.RLock()
	checks := make([]Check, 0, len(hc.checks))
	for _, check := range hc.checks {
		checks = append(checks, check)
	}
	hc.mu.RUnlock()

	checkStatuses := make([]CheckStatus, 0, len(checks))
	overallStatus := HealthyStatus

	for _, check := range checks {
		err := check.Check(ctx)
		status := HealthyStatus
		errStr := ""

		if err != nil {
			status = UnhealthyStatus
			errStr = err.Error()
			overallStatus = UnhealthyStatus
		}

		checkStatuses = append(checkStatuses, CheckStatus{
			Name:   check.Name(),
			Status: status,
			Error:  errStr,
		})
	}

	return &HealthStatus{
		Status:    overallStatus,
		Checks:    checkStatuses,
		Timestamp: time.Now(),
	}
}

// GetCheckStatus retrieves the status of a specific check.
func (hc *HealthChecker) GetCheckStatus(ctx context.Context, name string) (*CheckStatus, error) {
	hc.mu.RLock()
	check, exists := hc.checks[name]
	hc.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("check %q not found", name)
	}

	err := check.Check(ctx)
	status := HealthyStatus
	errStr := ""

	if err != nil {
		status = UnhealthyStatus
		errStr = err.Error()
	}

	return &CheckStatus{
		Name:   name,
		Status: status,
		Error:  errStr,
	}, nil
}

// HealthHTTPHandler returns an HTTP handler for the /health endpoint.
func (hc *HealthChecker) HealthHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		healthStatus := hc.CheckAll(ctx)

		w.Header().Set("Content-Type", "application/json")

		statusCode := http.StatusOK
		if healthStatus.Status != HealthyStatus {
			statusCode = http.StatusServiceUnavailable
		}
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(healthStatus); err != nil {
			hc.logger.Error("failed to encode health status", "error", err)
		}
	}
}

// ReadyHTTPHandler returns an HTTP handler for the /ready endpoint.
func (hc *HealthChecker) ReadyHTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		healthStatus := hc.CheckAll(ctx)

		w.Header().Set("Content-Type", "application/json")

		statusCode := http.StatusOK
		if healthStatus.Status != HealthyStatus {
			statusCode = http.StatusServiceUnavailable
		}
		w.WriteHeader(statusCode)

		if err := json.NewEncoder(w).Encode(healthStatus); err != nil {
			hc.logger.Error("failed to encode ready status", "error", err)
		}
	}
}

// Built-in health checks

// GenericCheck is a generic health check implementation.
type GenericCheck struct {
	name    string
	checkFn func(context.Context) error
}

// NewGenericCheck creates a new generic health check.
func NewGenericCheck(name string, checkFn func(context.Context) error) *GenericCheck {
	return &GenericCheck{
		name:    name,
		checkFn: checkFn,
	}
}

// Name returns the check name.
func (gc *GenericCheck) Name() string {
	return gc.name
}

// Check executes the check.
func (gc *GenericCheck) Check(ctx context.Context) error {
	return gc.checkFn(ctx)
}

// KafkaHealthCheck checks the health of Kafka connection.
type KafkaHealthCheck struct {
	name   string
	pingFn func(context.Context) error
}

// NewKafkaHealthCheck creates a new Kafka health check.
func NewKafkaHealthCheck(pingFn func(context.Context) error) *KafkaHealthCheck {
	return &KafkaHealthCheck{
		name:   "kafka",
		pingFn: pingFn,
	}
}

// Name returns the check name.
func (khc *KafkaHealthCheck) Name() string {
	return khc.name
}

// Check executes the Kafka health check.
func (khc *KafkaHealthCheck) Check(ctx context.Context) error {
	if khc.pingFn == nil {
		return fmt.Errorf("ping function not set")
	}
	return khc.pingFn(ctx)
}

// GRPCHealthCheck checks the health of gRPC server.
type GRPCHealthCheck struct {
	name   string
	pingFn func(context.Context) error
}

// NewGRPCHealthCheck creates a new gRPC health check.
func NewGRPCHealthCheck(pingFn func(context.Context) error) *GRPCHealthCheck {
	return &GRPCHealthCheck{
		name:   "grpc",
		pingFn: pingFn,
	}
}

// Name returns the check name.
func (ghc *GRPCHealthCheck) Name() string {
	return ghc.name
}

// Check executes the gRPC health check.
func (ghc *GRPCHealthCheck) Check(ctx context.Context) error {
	if ghc.pingFn == nil {
		return fmt.Errorf("ping function not set")
	}
	return ghc.pingFn(ctx)
}

// SimpleHealthCheck is a basic health check that always passes.
type SimpleHealthCheck struct {
	name string
}

// NewSimpleHealthCheck creates a new simple health check.
func NewSimpleHealthCheck(name string) *SimpleHealthCheck {
	return &SimpleHealthCheck{
		name: name,
	}
}

// Name returns the check name.
func (shc *SimpleHealthCheck) Name() string {
	return shc.name
}

// Check always returns nil (healthy).
func (shc *SimpleHealthCheck) Check(ctx context.Context) error {
	return nil
}
