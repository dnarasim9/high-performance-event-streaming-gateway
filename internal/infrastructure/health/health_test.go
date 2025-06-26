package health

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError}))
}

type MockCheck struct {
	name    string
	checkFn func(context.Context) error
}

func (m *MockCheck) Name() string {
	return m.name
}

func (m *MockCheck) Check(ctx context.Context) error {
	if m.checkFn != nil {
		return m.checkFn(ctx)
	}
	return nil
}

func TestHealthChecker_RegisterCheck_Success(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	check := &MockCheck{
		name: "test-check",
	}

	err := hc.RegisterCheck(check)
	if err != nil {
		t.Errorf("RegisterCheck() error = %v, want nil", err)
	}
}

func TestHealthChecker_RegisterCheck_NilCheck(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	err := hc.RegisterCheck(nil)
	if err == nil {
		t.Error("RegisterCheck() with nil check expected error, got nil")
	}
}

func TestHealthChecker_RegisterCheck_EmptyName(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	check := &MockCheck{
		name: "",
	}

	err := hc.RegisterCheck(check)
	if err == nil {
		t.Error("RegisterCheck() with empty name expected error, got nil")
	}
}

func TestHealthChecker_RegisterCheck_Duplicate(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	check := &MockCheck{
		name: "duplicate-check",
	}

	hc.RegisterCheck(check)

	// Try to register with same name
	err := hc.RegisterCheck(check)
	if err == nil {
		t.Error("RegisterCheck() with duplicate name expected error, got nil")
	}
}

func TestHealthChecker_CheckAll_AllHealthy(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	check1 := &MockCheck{
		name: "check1",
		checkFn: func(ctx context.Context) error {
			return nil
		},
	}

	check2 := &MockCheck{
		name: "check2",
		checkFn: func(ctx context.Context) error {
			return nil
		},
	}

	hc.RegisterCheck(check1)
	hc.RegisterCheck(check2)

	status := hc.CheckAll(context.Background())

	if status.Status != "healthy" {
		t.Errorf("CheckAll() status = %q, want healthy", status.Status)
	}

	if len(status.Checks) != 2 {
		t.Errorf("CheckAll() returned %d checks, want 2", len(status.Checks))
	}

	for _, check := range status.Checks {
		if check.Status != "healthy" {
			t.Errorf("CheckAll() check %s status = %q, want healthy", check.Name, check.Status)
		}
		if check.Error != "" {
			t.Errorf("CheckAll() check %s has error, want empty", check.Name)
		}
	}
}

func TestHealthChecker_CheckAll_OneUnhealthy(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	check1 := &MockCheck{
		name: "healthy-check",
		checkFn: func(ctx context.Context) error {
			return nil
		},
	}

	check2 := &MockCheck{
		name: "unhealthy-check",
		checkFn: func(ctx context.Context) error {
			return errors.New("service down")
		},
	}

	hc.RegisterCheck(check1)
	hc.RegisterCheck(check2)

	status := hc.CheckAll(context.Background())

	if status.Status != "unhealthy" {
		t.Errorf("CheckAll() status = %q, want unhealthy", status.Status)
	}

	unhealthyCount := 0
	for _, check := range status.Checks {
		if check.Status == "unhealthy" {
			unhealthyCount++
			if check.Error == "" {
				t.Errorf("CheckAll() unhealthy check should have error message")
			}
		}
	}

	if unhealthyCount != 1 {
		t.Errorf("CheckAll() found %d unhealthy checks, want 1", unhealthyCount)
	}
}

func TestHealthChecker_CheckAll_AllUnhealthy(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	check1 := &MockCheck{
		name: "check1",
		checkFn: func(ctx context.Context) error {
			return errors.New("error 1")
		},
	}

	check2 := &MockCheck{
		name: "check2",
		checkFn: func(ctx context.Context) error {
			return errors.New("error 2")
		},
	}

	hc.RegisterCheck(check1)
	hc.RegisterCheck(check2)

	status := hc.CheckAll(context.Background())

	if status.Status != "unhealthy" {
		t.Errorf("CheckAll() status = %q, want unhealthy", status.Status)
	}

	for _, check := range status.Checks {
		if check.Status != "unhealthy" {
			t.Errorf("CheckAll() check %s status = %q, want unhealthy", check.Name, check.Status)
		}
	}
}

func TestHealthChecker_CheckAll_NoChecks(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	status := hc.CheckAll(context.Background())

	if status.Status != "healthy" {
		t.Errorf("CheckAll() with no checks status = %q, want healthy", status.Status)
	}

	if len(status.Checks) != 0 {
		t.Errorf("CheckAll() returned %d checks, want 0", len(status.Checks))
	}
}

func TestHealthChecker_GetCheckStatus_Success(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	check := &MockCheck{
		name: "test-check",
		checkFn: func(ctx context.Context) error {
			return nil
		},
	}

	hc.RegisterCheck(check)

	status, err := hc.GetCheckStatus(context.Background(), "test-check")
	if err != nil {
		t.Errorf("GetCheckStatus() error = %v, want nil", err)
	}

	if status.Status != "healthy" {
		t.Errorf("GetCheckStatus() status = %q, want healthy", status.Status)
	}

	if status.Name != "test-check" {
		t.Errorf("GetCheckStatus() name = %q, want test-check", status.Name)
	}
}

func TestHealthChecker_GetCheckStatus_NotFound(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	_, err := hc.GetCheckStatus(context.Background(), "non-existent")
	if err == nil {
		t.Error("GetCheckStatus() for non-existent check expected error, got nil")
	}
}

func TestHealthChecker_GetCheckStatus_Unhealthy(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	check := &MockCheck{
		name: "failing-check",
		checkFn: func(ctx context.Context) error {
			return errors.New("check failed")
		},
	}

	hc.RegisterCheck(check)

	status, err := hc.GetCheckStatus(context.Background(), "failing-check")
	if err != nil {
		t.Errorf("GetCheckStatus() error = %v, want nil", err)
	}

	if status.Status != "unhealthy" {
		t.Errorf("GetCheckStatus() status = %q, want unhealthy", status.Status)
	}

	if status.Error == "" {
		t.Error("GetCheckStatus() unhealthy check should have error message")
	}
}

func TestGenericCheck(t *testing.T) {
	checkFn := func(ctx context.Context) error {
		return nil
	}

	check := NewGenericCheck("generic-check", checkFn)

	if check.Name() != "generic-check" {
		t.Errorf("GenericCheck Name() = %q, want generic-check", check.Name())
	}

	err := check.Check(context.Background())
	if err != nil {
		t.Errorf("GenericCheck Check() error = %v, want nil", err)
	}
}

func TestKafkaHealthCheck(t *testing.T) {
	pingFn := func(ctx context.Context) error {
		return nil
	}

	check := NewKafkaHealthCheck(pingFn)

	if check.Name() != "kafka" {
		t.Errorf("KafkaHealthCheck Name() = %q, want kafka", check.Name())
	}

	err := check.Check(context.Background())
	if err != nil {
		t.Errorf("KafkaHealthCheck Check() error = %v, want nil", err)
	}
}

func TestKafkaHealthCheck_NoFunction(t *testing.T) {
	check := NewKafkaHealthCheck(nil)

	err := check.Check(context.Background())
	if err == nil {
		t.Error("KafkaHealthCheck Check() with nil function expected error, got nil")
	}
}

func TestGRPCHealthCheck(t *testing.T) {
	pingFn := func(ctx context.Context) error {
		return nil
	}

	check := NewGRPCHealthCheck(pingFn)

	if check.Name() != "grpc" {
		t.Errorf("GRPCHealthCheck Name() = %q, want grpc", check.Name())
	}

	err := check.Check(context.Background())
	if err != nil {
		t.Errorf("GRPCHealthCheck Check() error = %v, want nil", err)
	}
}

func TestGRPCHealthCheck_NoFunction(t *testing.T) {
	check := NewGRPCHealthCheck(nil)

	err := check.Check(context.Background())
	if err == nil {
		t.Error("GRPCHealthCheck Check() with nil function expected error, got nil")
	}
}

func TestSimpleHealthCheck(t *testing.T) {
	check := NewSimpleHealthCheck("simple-check")

	if check.Name() != "simple-check" {
		t.Errorf("SimpleHealthCheck Name() = %q, want simple-check", check.Name())
	}

	err := check.Check(context.Background())
	if err != nil {
		t.Errorf("SimpleHealthCheck Check() error = %v, want nil", err)
	}
}

func TestHealthStatus_Timestamp(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	status := hc.CheckAll(context.Background())

	if status.Timestamp.IsZero() {
		t.Error("HealthStatus Timestamp should not be zero")
	}
}

func TestHealthChecker_ConcurrentRegisterAndCheck(t *testing.T) {
	hc := NewHealthChecker(newTestLogger())

	// Register initial checks
	for i := 0; i < 3; i++ {
		check := &MockCheck{
			name: "check-" + string(rune('0'+i)),
			checkFn: func(ctx context.Context) error {
				return nil
			},
		}
		hc.RegisterCheck(check)
	}

	// Check concurrently
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			hc.CheckAll(context.Background())
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}
