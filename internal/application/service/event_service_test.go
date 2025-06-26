package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/repository"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// Mock implementations
type MockEventStore struct {
	StoreFunc func(ctx context.Context, event *entity.Event) error
	GetFunc   func(ctx context.Context, id string) (*entity.Event, error)
	QueryFunc func(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error)
}

func (m *MockEventStore) Store(ctx context.Context, event *entity.Event) error {
	if m.StoreFunc != nil {
		return m.StoreFunc(ctx, event)
	}
	return nil
}

func (m *MockEventStore) Get(ctx context.Context, id string) (*entity.Event, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockEventStore) Query(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
	if m.QueryFunc != nil {
		return m.QueryFunc(ctx, filter, timeRange)
	}
	return nil, nil
}

type MockMessageBroker struct {
	PublishFunc      func(ctx context.Context, event *entity.Event, topic string) error
	PublishBatchFunc func(ctx context.Context, events []*entity.Event, topic string) error
	SubscribeFunc    func(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error)
	UnsubscribeFunc  func() error
	CloseFunc        func() error
}

func (m *MockMessageBroker) Publish(ctx context.Context, event *entity.Event, topic string) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(ctx, event, topic)
	}
	return nil
}

func (m *MockMessageBroker) PublishBatch(ctx context.Context, events []*entity.Event, topic string) error {
	if m.PublishBatchFunc != nil {
		return m.PublishBatchFunc(ctx, events, topic)
	}
	return nil
}

func (m *MockMessageBroker) Subscribe(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
	if m.SubscribeFunc != nil {
		return m.SubscribeFunc(ctx, topic, group, filter)
	}
	ch := make(chan *entity.Event)
	close(ch)
	return ch, nil
}

func (m *MockMessageBroker) Unsubscribe() error {
	if m.UnsubscribeFunc != nil {
		return m.UnsubscribeFunc()
	}
	return nil
}

func (m *MockMessageBroker) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

type MockSchemaValidator struct {
	ValidateFunc func(ctx context.Context, schemaID string, data []byte) error
}

func (m *MockSchemaValidator) Validate(ctx context.Context, schemaID string, data []byte) error {
	if m.ValidateFunc != nil {
		return m.ValidateFunc(ctx, schemaID, data)
	}
	return nil
}

type MockMetricsCollector struct {
	RecordIngestionFunc    func(eventCount int, durationMS float64)
	RecordPublishFunc      func(eventCount int, durationMS float64)
	RecordSubscriptionFunc func(consumerID, groupID string)
	RecordLatencyFunc      func(operation string, durationMS float64)
	RecordErrorFunc        func(operation, errorType string)
}

func (m *MockMetricsCollector) RecordIngestion(eventCount int, durationMS float64) {
	if m.RecordIngestionFunc != nil {
		m.RecordIngestionFunc(eventCount, durationMS)
	}
}

func (m *MockMetricsCollector) RecordPublish(eventCount int, durationMS float64) {
	if m.RecordPublishFunc != nil {
		m.RecordPublishFunc(eventCount, durationMS)
	}
}

func (m *MockMetricsCollector) RecordSubscription(consumerID, groupID string) {
	if m.RecordSubscriptionFunc != nil {
		m.RecordSubscriptionFunc(consumerID, groupID)
	}
}

func (m *MockMetricsCollector) RecordLatency(operation string, durationMS float64) {
	if m.RecordLatencyFunc != nil {
		m.RecordLatencyFunc(operation, durationMS)
	}
}

func (m *MockMetricsCollector) RecordError(operation, errorType string) {
	if m.RecordErrorFunc != nil {
		m.RecordErrorFunc(operation, errorType)
	}
}

func newTestLogger() *slog.Logger {
	// Use a noop logger for tests
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEventService_IngestEvent_Success(t *testing.T) {
	tests := []struct {
		name string
		req  *dto.IngestEventRequest
	}{
		{
			name: "basic event",
			req: &dto.IngestEventRequest{
				Source:          "test-source",
				Type:            "test.created",
				Subject:         "/test/1",
				Data:            []byte(`{"id": 1}`),
				DataContentType: "application/json",
			},
		},
		{
			name: "event with metadata",
			req: &dto.IngestEventRequest{
				Source:          "api",
				Type:            "user.registered",
				Subject:         "/users/123",
				Data:            []byte(`{"username": "testuser"}`),
				DataContentType: "application/json",
				Metadata: map[string]string{
					"env": "production",
				},
			},
		},
		{
			name: "event with correlation",
			req: &dto.IngestEventRequest{
				Source:          "service",
				Type:            "order.placed",
				Subject:         "/orders/456",
				Data:            []byte(`{"order_id": 456}`),
				DataContentType: "application/json",
				CorrelationID:   "corr-123",
				CausationID:     "cause-456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &MockEventStore{}
			broker := &MockMessageBroker{}
			validator := &MockSchemaValidator{}
			metrics := &MockMetricsCollector{}

			service := NewEventService(store, broker, validator, metrics, newTestLogger())

			eventID, err := service.IngestEvent(context.Background(), tt.req)
			if err != nil {
				t.Errorf("IngestEvent() error = %v, want nil", err)
			}

			if eventID == "" {
				t.Error("IngestEvent() returned empty event ID")
			}
		})
	}
}

func TestEventService_IngestEvent_ValidationError(t *testing.T) {
	tests := []struct {
		name string
		req  *dto.IngestEventRequest
	}{
		{
			name: "empty source",
			req: &dto.IngestEventRequest{
				Source:          "",
				Type:            "test.created",
				Subject:         "/test/1",
				Data:            []byte(`{}`),
				DataContentType: "application/json",
			},
		},
		{
			name: "empty type",
			req: &dto.IngestEventRequest{
				Source:          "source",
				Type:            "",
				Subject:         "/test/1",
				Data:            []byte(`{}`),
				DataContentType: "application/json",
			},
		},
		{
			name: "empty data",
			req: &dto.IngestEventRequest{
				Source:          "source",
				Type:            "test.created",
				Subject:         "/test/1",
				Data:            []byte{},
				DataContentType: "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &MockEventStore{}
			broker := &MockMessageBroker{}
			validator := &MockSchemaValidator{}
			metrics := &MockMetricsCollector{}

			service := NewEventService(store, broker, validator, metrics, newTestLogger())

			_, err := service.IngestEvent(context.Background(), tt.req)
			if err == nil {
				t.Error("IngestEvent() expected error, got nil")
			}
		})
	}
}

func TestEventService_IngestEvent_SchemaValidationFailure(t *testing.T) {
	req := &dto.IngestEventRequest{
		Source:          "test",
		Type:            "test.created",
		Subject:         "/test/1",
		Data:            []byte(`{}`),
		DataContentType: "application/json",
		SchemaURL:       "schema-123",
	}

	store := &MockEventStore{}
	broker := &MockMessageBroker{}
	validator := &MockSchemaValidator{
		ValidateFunc: func(ctx context.Context, schemaID string, data []byte) error {
			return errors.New("schema validation failed")
		},
	}
	metrics := &MockMetricsCollector{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())

	_, err := service.IngestEvent(context.Background(), req)
	if err == nil {
		t.Error("IngestEvent() expected error, got nil")
	}
	if !errors.Is(err, errors.New("")) && err.Error() != "schema validation failed: schema validation failed" {
		t.Logf("got error: %v", err)
	}
}

func TestEventService_IngestEvent_StoreFailure(t *testing.T) {
	req := &dto.IngestEventRequest{
		Source:          "test",
		Type:            "test.created",
		Subject:         "/test/1",
		Data:            []byte(`{}`),
		DataContentType: "application/json",
	}

	store := &MockEventStore{
		StoreFunc: func(ctx context.Context, event *entity.Event) error {
			return errors.New("store failure")
		},
	}
	broker := &MockMessageBroker{}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())

	_, err := service.IngestEvent(context.Background(), req)
	if err == nil {
		t.Error("IngestEvent() expected error, got nil")
	}
}

func TestEventService_IngestEvent_PublishFailure(t *testing.T) {
	req := &dto.IngestEventRequest{
		Source:          "test",
		Type:            "test.created",
		Subject:         "/test/1",
		Data:            []byte(`{}`),
		DataContentType: "application/json",
	}

	store := &MockEventStore{}
	broker := &MockMessageBroker{
		PublishFunc: func(ctx context.Context, event *entity.Event, topic string) error {
			return errors.New("publish failure")
		},
	}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())

	_, err := service.IngestEvent(context.Background(), req)
	if err == nil {
		t.Error("IngestEvent() expected error, got nil")
	}
}

func TestEventService_IngestEventBatch_Success(t *testing.T) {
	requests := []*dto.IngestEventRequest{
		{
			Source:          "test",
			Type:            "test.created",
			Subject:         "/test/1",
			Data:            []byte(`{"id": 1}`),
			DataContentType: "application/json",
		},
		{
			Source:          "test",
			Type:            "test.created",
			Subject:         "/test/2",
			Data:            []byte(`{"id": 2}`),
			DataContentType: "application/json",
		},
		{
			Source:          "test",
			Type:            "test.created",
			Subject:         "/test/3",
			Data:            []byte(`{"id": 3}`),
			DataContentType: "application/json",
		},
	}

	store := &MockEventStore{}
	broker := &MockMessageBroker{}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())

	eventIDs, err := service.IngestEventBatch(context.Background(), requests)
	if err != nil {
		t.Errorf("IngestEventBatch() error = %v, want nil", err)
	}

	if len(eventIDs) != len(requests) {
		t.Errorf("IngestEventBatch() returned %d IDs, want %d", len(eventIDs), len(requests))
	}
}

func TestEventService_IngestEventBatch_EmptyBatch(t *testing.T) {
	store := &MockEventStore{}
	broker := &MockMessageBroker{}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())

	eventIDs, err := service.IngestEventBatch(context.Background(), []*dto.IngestEventRequest{})
	if err != nil {
		t.Errorf("IngestEventBatch() error = %v, want nil", err)
	}

	if len(eventIDs) != 0 {
		t.Errorf("IngestEventBatch() returned %d IDs, want 0", len(eventIDs))
	}
}

func TestEventService_IngestEventBatch_ValidationError(t *testing.T) {
	requests := []*dto.IngestEventRequest{
		{
			Source:          "test",
			Type:            "test.created",
			Subject:         "/test/1",
			Data:            []byte(`{}`),
			DataContentType: "application/json",
		},
		{
			Source:          "",
			Type:            "test.created",
			Subject:         "/test/2",
			Data:            []byte(`{}`),
			DataContentType: "application/json",
		},
	}

	store := &MockEventStore{}
	broker := &MockMessageBroker{}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())

	_, err := service.IngestEventBatch(context.Background(), requests)
	if err == nil {
		t.Error("IngestEventBatch() expected error, got nil")
	}
}

func TestEventService_IngestEventBatch_StoreFailure(t *testing.T) {
	requests := []*dto.IngestEventRequest{
		{
			Source:          "test",
			Type:            "test.created",
			Subject:         "/test/1",
			Data:            []byte(`{}`),
			DataContentType: "application/json",
		},
	}

	store := &MockEventStore{
		StoreFunc: func(ctx context.Context, event *entity.Event) error {
			return errors.New("store failure")
		},
	}
	broker := &MockMessageBroker{}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())

	_, err := service.IngestEventBatch(context.Background(), requests)
	if err == nil {
		t.Error("IngestEventBatch() expected error, got nil")
	}
}

func TestEventService_IngestEventBatch_PublishFailure(t *testing.T) {
	requests := []*dto.IngestEventRequest{
		{
			Source:          "test",
			Type:            "test.created",
			Subject:         "/test/1",
			Data:            []byte(`{}`),
			DataContentType: "application/json",
		},
	}

	store := &MockEventStore{}
	broker := &MockMessageBroker{
		PublishFunc: func(ctx context.Context, event *entity.Event, topic string) error {
			return errors.New("publish failure")
		},
	}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())

	_, err := service.IngestEventBatch(context.Background(), requests)
	if err == nil {
		t.Error("IngestEventBatch() expected error, got nil")
	}
}

func TestEventService_IngestEventBatch_SchemaValidationFailure(t *testing.T) {
	requests := []*dto.IngestEventRequest{
		{
			Source:          "test",
			Type:            "test.created",
			Subject:         "/test/1",
			Data:            []byte(`{}`),
			DataContentType: "application/json",
			SchemaURL:       "schema-123",
		},
	}

	store := &MockEventStore{}
	broker := &MockMessageBroker{}
	validator := &MockSchemaValidator{
		ValidateFunc: func(ctx context.Context, schemaID string, data []byte) error {
			return errors.New("invalid schema")
		},
	}
	metrics := &MockMetricsCollector{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())

	_, err := service.IngestEventBatch(context.Background(), requests)
	if err == nil {
		t.Error("IngestEventBatch() expected error, got nil")
	}
}

func TestEventService_MetricsRecording(t *testing.T) {
	req := &dto.IngestEventRequest{
		Source:          "test",
		Type:            "test.created",
		Subject:         "/test/1",
		Data:            []byte(`{}`),
		DataContentType: "application/json",
	}

	recordedMetrics := []struct {
		operation string
		eventType string
	}{}

	metrics := &MockMetricsCollector{
		RecordIngestionFunc: func(eventCount int, durationMS float64) {
			if eventCount != 1 {
				t.Errorf("RecordIngestion() eventCount = %d, want 1", eventCount)
			}
		},
		RecordErrorFunc: func(operation, errorType string) {
			recordedMetrics = append(recordedMetrics, struct {
				operation string
				eventType string
			}{operation, errorType})
		},
	}

	store := &MockEventStore{}
	broker := &MockMessageBroker{}
	validator := &MockSchemaValidator{}

	service := NewEventService(store, broker, validator, metrics, newTestLogger())
	service.IngestEvent(context.Background(), req)
}
