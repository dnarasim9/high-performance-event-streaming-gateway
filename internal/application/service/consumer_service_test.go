package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// Mock implementations for ConsumerService testing

type MockConsumerRepository struct {
	SaveFunc   func(ctx context.Context, consumer *entity.Consumer) error
	GetFunc    func(ctx context.Context, consumerID string) (*entity.Consumer, error)
	ListFunc   func(ctx context.Context, groupID string) ([]*entity.Consumer, error)
	DeleteFunc func(ctx context.Context, consumerID string) error
}

func (m *MockConsumerRepository) Save(ctx context.Context, consumer *entity.Consumer) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, consumer)
	}
	return nil
}

func (m *MockConsumerRepository) Get(ctx context.Context, consumerID string) (*entity.Consumer, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, consumerID)
	}
	return nil, nil
}

func (m *MockConsumerRepository) List(ctx context.Context, groupID string) ([]*entity.Consumer, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, groupID)
	}
	return nil, nil
}

func (m *MockConsumerRepository) Delete(ctx context.Context, consumerID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, consumerID)
	}
	return nil
}

// TestConsumerService_Register_Success tests successful consumer registration
func TestConsumerService_Register_Success(t *testing.T) {
	tests := []struct {
		name string
		req  *dto.RegisterConsumerRequest
	}{
		{
			name: "basic consumer registration",
			req: &dto.RegisterConsumerRequest{
				Name:    "test-consumer",
				GroupID: "test-group",
			},
		},
		{
			name: "consumer with filter config",
			req: &dto.RegisterConsumerRequest{
				Name:    "filtered-consumer",
				GroupID: "group-1",
				FilterConfig: &dto.EventFilterDTO{
					TypePattern: "user.created",
				},
			},
		},
		{
			name: "consumer with transformation",
			req: &dto.RegisterConsumerRequest{
				Name:    "transformed-consumer",
				GroupID: "group-2",
				TransformConfig: &dto.TransformationConfigDTO{
					Type:       "None",
					Expression: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockConsumerRepository{
				SaveFunc: func(ctx context.Context, consumer *entity.Consumer) error {
					return nil
				},
			}
			metrics := &MockMetricsCollector{}
			service := NewConsumerService(repo, metrics, newTestLogger())

			consumerID, err := service.Register(context.Background(), tt.req)

			if err != nil {
				t.Errorf("Register() error = %v, want nil", err)
			}
			if consumerID == "" {
				t.Error("Register() returned empty consumer ID")
			}
		})
	}
}

// TestConsumerService_Register_RepositoryErrorOnActivation tests registration failure during activation
func TestConsumerService_Register_RepositoryErrorOnActivation(t *testing.T) {
	repo := &MockConsumerRepository{
		SaveFunc: func(ctx context.Context, consumer *entity.Consumer) error {
			return errors.New("activation error")
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	req := &dto.RegisterConsumerRequest{
		Name:    "test-consumer",
		GroupID: "group-1",
	}

	_, err := service.Register(context.Background(), req)
	if err == nil {
		t.Error("Register() with activation error expected error, got nil")
	}
}

// TestConsumerService_Register_RepositoryError tests registration with repository error
func TestConsumerService_Register_RepositoryError(t *testing.T) {
	repo := &MockConsumerRepository{
		SaveFunc: func(ctx context.Context, consumer *entity.Consumer) error {
			return errors.New("storage error")
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	req := &dto.RegisterConsumerRequest{
		Name:    "test-consumer",
		GroupID: "group-1",
	}

	_, err := service.Register(context.Background(), req)
	if err == nil {
		t.Error("Register() with repository error expected error, got nil")
	}
}

// TestConsumerService_Deregister_Success tests successful consumer deregistration
func TestConsumerService_Deregister_Success(t *testing.T) {
	consumer, err := entity.NewConsumer(
		"consumer-1",
		"test",
		"group-1",
		valueobject.EventFilter{},
		valueobject.Transformation{Type: valueobject.TransformationTypeNone},
	)
	if err != nil {
		t.Fatalf("Failed to create test consumer: %v", err)
	}

	repo := &MockConsumerRepository{
		GetFunc: func(ctx context.Context, consumerID string) (*entity.Consumer, error) {
			return consumer, nil
		},
		DeleteFunc: func(ctx context.Context, consumerID string) error {
			return nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	err = service.Deregister(context.Background(), "consumer-1")
	if err != nil {
		t.Errorf("Deregister() error = %v, want nil", err)
	}
}

// TestConsumerService_Deregister_NotFound tests deregistration of non-existent consumer
func TestConsumerService_Deregister_NotFound(t *testing.T) {
	repo := &MockConsumerRepository{
		GetFunc: func(ctx context.Context, consumerID string) (*entity.Consumer, error) {
			return nil, errors.New("not found")
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	err := service.Deregister(context.Background(), "non-existent")
	if err == nil {
		t.Error("Deregister() for non-existent consumer expected error, got nil")
	}
}

// TestConsumerService_Deregister_DeleteError tests deregistration with deletion error
func TestConsumerService_Deregister_DeleteError(t *testing.T) {
	consumer, err := entity.NewConsumer(
		"consumer-1",
		"test",
		"group-1",
		valueobject.EventFilter{},
		valueobject.Transformation{Type: valueobject.TransformationTypeNone},
	)
	if err != nil {
		t.Fatalf("Failed to create test consumer: %v", err)
	}

	repo := &MockConsumerRepository{
		GetFunc: func(ctx context.Context, consumerID string) (*entity.Consumer, error) {
			return consumer, nil
		},
		DeleteFunc: func(ctx context.Context, consumerID string) error {
			return errors.New("deletion failed")
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	err = service.Deregister(context.Background(), "consumer-1")
	if err == nil {
		t.Error("Deregister() with deletion error expected error, got nil")
	}
}

// TestConsumerService_Get_Success tests successful consumer retrieval
func TestConsumerService_Get_Success(t *testing.T) {
	consumer, err := entity.NewConsumer(
		"consumer-1",
		"test-consumer",
		"group-1",
		valueobject.EventFilter{},
		valueobject.Transformation{Type: valueobject.TransformationTypeNone},
	)
	if err != nil {
		t.Fatalf("Failed to create test consumer: %v", err)
	}

	repo := &MockConsumerRepository{
		GetFunc: func(ctx context.Context, consumerID string) (*entity.Consumer, error) {
			return consumer, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	result, err := service.Get(context.Background(), "consumer-1")
	if err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	}
	if result == nil {
		t.Error("Get() returned nil")
	}
	if result.Name != "test-consumer" {
		t.Errorf("Get() name = %s, want test-consumer", result.Name)
	}
}

// TestConsumerService_Get_NotFound tests retrieval of non-existent consumer
func TestConsumerService_Get_NotFound(t *testing.T) {
	repo := &MockConsumerRepository{
		GetFunc: func(ctx context.Context, consumerID string) (*entity.Consumer, error) {
			return nil, errors.New("not found")
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	_, err := service.Get(context.Background(), "non-existent")
	if err == nil {
		t.Error("Get() for non-existent consumer expected error, got nil")
	}
}

// TestConsumerService_List_Success tests successful consumer list retrieval
func TestConsumerService_List_Success(t *testing.T) {
	consumers := make([]*entity.Consumer, 2)
	for i := 0; i < 2; i++ {
		consumer, err := entity.NewConsumer(
			"consumer-"+string(rune(i+'1')),
			"test-consumer-"+string(rune(i+'1')),
			"group-1",
			valueobject.EventFilter{},
			valueobject.Transformation{Type: valueobject.TransformationTypeNone},
		)
		if err != nil {
			t.Fatalf("Failed to create test consumer: %v", err)
		}
		consumers[i] = consumer
	}

	repo := &MockConsumerRepository{
		ListFunc: func(ctx context.Context, groupID string) ([]*entity.Consumer, error) {
			return consumers, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	result, err := service.List(context.Background(), "group-1")
	if err != nil {
		t.Errorf("List() error = %v, want nil", err)
	}
	if len(result) != 2 {
		t.Errorf("List() returned %d consumers, want 2", len(result))
	}
}

// TestConsumerService_List_Error tests list retrieval with error
func TestConsumerService_List_Error(t *testing.T) {
	repo := &MockConsumerRepository{
		ListFunc: func(ctx context.Context, groupID string) ([]*entity.Consumer, error) {
			return nil, errors.New("list error")
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	_, err := service.List(context.Background(), "group-1")
	if err == nil {
		t.Error("List() with error expected error, got nil")
	}
}

// TestConsumerService_List_Empty tests list retrieval with empty result
func TestConsumerService_List_Empty(t *testing.T) {
	repo := &MockConsumerRepository{
		ListFunc: func(ctx context.Context, groupID string) ([]*entity.Consumer, error) {
			return []*entity.Consumer{}, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	result, err := service.List(context.Background(), "group-1")
	if err != nil {
		t.Errorf("List() error = %v, want nil", err)
	}
	if len(result) != 0 {
		t.Errorf("List() returned %d consumers, want 0", len(result))
	}
}

// TestConsumerService_Register_WithComplexFilter tests registration with complex filter
func TestConsumerService_Register_WithComplexFilter(t *testing.T) {
	repo := &MockConsumerRepository{
		SaveFunc: func(ctx context.Context, consumer *entity.Consumer) error {
			return nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewConsumerService(repo, metrics, newTestLogger())

	req := &dto.RegisterConsumerRequest{
		Name:    "complex-consumer",
		GroupID: "group-1",
		FilterConfig: &dto.EventFilterDTO{
			TypePattern:    "user.created",
			SourcePattern:  "api",
			SubjectPattern: "/users",
			MinPriority:    int(entity.PriorityHigh),
		},
	}

	consumerID, err := service.Register(context.Background(), req)
	if err != nil {
		t.Errorf("Register() error = %v, want nil", err)
	}
	if consumerID == "" {
		t.Error("Register() returned empty consumer ID")
	}
}

// TestConsumerService_RegisterMetricsRecorded tests that metrics are recorded during registration
func TestConsumerService_RegisterMetricsRecorded(t *testing.T) {
	repo := &MockConsumerRepository{
		SaveFunc: func(ctx context.Context, consumer *entity.Consumer) error {
			return nil
		},
	}

	metricsRecorded := false
	metrics := &MockMetricsCollector{
		RecordLatencyFunc: func(operation string, durationMS float64) {
			if operation == "register" {
				metricsRecorded = true
			}
		},
	}
	service := NewConsumerService(repo, metrics, newTestLogger())

	req := &dto.RegisterConsumerRequest{
		Name:    "test-consumer",
		GroupID: "group-1",
	}

	_, err := service.Register(context.Background(), req)
	if err != nil {
		t.Errorf("Register() error = %v, want nil", err)
	}
	if !metricsRecorded {
		t.Error("Register() did not record latency metrics")
	}
}

// TestConsumerService_DeregisterMetricsRecorded tests that metrics are recorded during deregistration
func TestConsumerService_DeregisterMetricsRecorded(t *testing.T) {
	consumer, err := entity.NewConsumer(
		"consumer-1",
		"test",
		"group-1",
		valueobject.EventFilter{},
		valueobject.Transformation{Type: valueobject.TransformationTypeNone},
	)
	if err != nil {
		t.Fatalf("Failed to create test consumer: %v", err)
	}

	repo := &MockConsumerRepository{
		GetFunc: func(ctx context.Context, consumerID string) (*entity.Consumer, error) {
			return consumer, nil
		},
		DeleteFunc: func(ctx context.Context, consumerID string) error {
			return nil
		},
	}

	metricsRecorded := false
	metrics := &MockMetricsCollector{
		RecordLatencyFunc: func(operation string, durationMS float64) {
			if operation == "deregister" {
				metricsRecorded = true
			}
		},
	}
	service := NewConsumerService(repo, metrics, newTestLogger())

	err = service.Deregister(context.Background(), "consumer-1")
	if err != nil {
		t.Errorf("Deregister() error = %v, want nil", err)
	}
	if !metricsRecorded {
		t.Error("Deregister() did not record latency metrics")
	}
}
