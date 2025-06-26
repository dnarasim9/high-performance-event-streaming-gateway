package benchmark

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/service"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/repository"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/config"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/ratelimiter"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/transformer"
	"github.com/dheemanth-hn/event-streaming-gateway/pkg/circuitbreaker"
)

// Mock implementations for benchmarking
type BenchEventStore struct{}

func (b *BenchEventStore) Store(ctx context.Context, event *entity.Event) error {
	return nil
}

func (b *BenchEventStore) Get(ctx context.Context, id string) (*entity.Event, error) {
	return nil, nil
}

func (b *BenchEventStore) Query(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
	return nil, nil
}

type BenchMessageBroker struct{}

func (b *BenchMessageBroker) Publish(ctx context.Context, event *entity.Event, topic string) error {
	return nil
}

func (b *BenchMessageBroker) PublishBatch(ctx context.Context, events []*entity.Event, topic string) error {
	return nil
}

func (b *BenchMessageBroker) Subscribe(ctx context.Context, topic, group string, filter valueobject.EventFilter) (<-chan *entity.Event, error) {
	ch := make(chan *entity.Event)
	close(ch)
	return ch, nil
}

func (b *BenchMessageBroker) Unsubscribe() error {
	return nil
}

func (b *BenchMessageBroker) Close() error {
	return nil
}

type BenchSchemaValidator struct{}

func (b *BenchSchemaValidator) Validate(ctx context.Context, schemaID string, data []byte) error {
	return nil
}

type BenchMetricsCollector struct{}

func (b *BenchMetricsCollector) RecordIngestion(eventCount int, durationMS float64) {}
func (b *BenchMetricsCollector) RecordPublish(eventCount int, durationMS float64)   {}
func (b *BenchMetricsCollector) RecordSubscription(consumerID, groupID string)      {}
func (b *BenchMetricsCollector) RecordLatency(operation string, durationMS float64) {}
func (b *BenchMetricsCollector) RecordError(operation, errorType string)            {}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

// BenchmarkEventCreationNew benchmarks event entity creation
func BenchmarkEventCreationNew(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		entity.NewEvent(
			"event-id",
			"test-source",
			"test.event",
			"test/subject",
			[]byte(`{"key": "value", "nested": {"field": "data"}}`),
			"application/json",
		)
	}
}

// BenchmarkEventIngestion benchmarks the full event ingestion service flow
func BenchmarkEventIngestion(b *testing.B) {
	store := &BenchEventStore{}
	broker := &BenchMessageBroker{}
	validator := &BenchSchemaValidator{}
	metrics := &BenchMetricsCollector{}

	svc := service.NewEventService(store, broker, validator, metrics, newTestLogger())

	request := &dto.IngestEventRequest{
		Source:          "benchmark-source",
		Type:            "benchmark.event",
		Subject:         "/benchmark/test",
		Data:            []byte(`{"benchmark": true, "iterations": 1000}`),
		DataContentType: "application/json",
		Metadata: map[string]string{
			"env":    "benchmark",
			"region": "local",
		},
	}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		svc.IngestEvent(ctx, request)
	}
}

// BenchmarkEventIngestionBatch benchmarks batch event ingestion
func BenchmarkEventIngestionBatch(b *testing.B) {
	store := &BenchEventStore{}
	broker := &BenchMessageBroker{}
	validator := &BenchSchemaValidator{}
	metrics := &BenchMetricsCollector{}

	svc := service.NewEventService(store, broker, validator, metrics, newTestLogger())

	requests := make([]*dto.IngestEventRequest, 10)
	for i := 0; i < 10; i++ {
		requests[i] = &dto.IngestEventRequest{
			Source:          "benchmark-source",
			Type:            "benchmark.event",
			Subject:         "/benchmark/test",
			Data:            []byte(`{"index": ` + string(rune('0'+i)) + `}`),
			DataContentType: "application/json",
		}
	}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		svc.IngestEventBatch(ctx, requests)
	}
}

// BenchmarkJQTransformation benchmarks JQ transformation engine
func BenchmarkJQTransformation(b *testing.B) {
	engine := transformer.NewTransformationEngine(1000000)

	event, _ := entity.NewEvent(
		"event-1",
		"test-source",
		"test.event",
		"test/subject",
		[]byte(`{"name": "test", "value": 42, "nested": {"field": "data"}}`),
		"application/json",
	)

	transformation := &transformer.Transformation{
		Type:   transformer.TransformationTypeJQ,
		Script: ". | {id, source, type, subject, data, dataContentType}",
	}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.Transform(ctx, event, transformation)
	}
}

// BenchmarkCELTransformation benchmarks CEL transformation engine
func BenchmarkCELTransformation(b *testing.B) {
	engine := transformer.NewTransformationEngine(1000000)

	event, _ := entity.NewEvent(
		"event-1",
		"test-source",
		"test.event",
		"test/subject",
		[]byte(`{"name": "test", "value": 42}`),
		"application/json",
	)

	transformation := &transformer.Transformation{
		Type:   transformer.TransformationTypeCEL,
		Script: `event`,
	}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.Transform(ctx, event, transformation)
	}
}

// BenchmarkTemplateTransformation benchmarks template transformation engine
func BenchmarkTemplateTransformation(b *testing.B) {
	engine := transformer.NewTransformationEngine(1000000)

	event, _ := entity.NewEvent(
		"event-1",
		"test-source",
		"test.event",
		"test/subject",
		[]byte(`{}`),
		"application/json",
	)

	transformation := &transformer.Transformation{
		Type:   transformer.TransformationTypeTemplate,
		Script: `{"id": "{{.ID}}", "source": "{{.Source}}", "type": "{{.Type}}", "subject": "{{.Subject}}", "dataContentType": "{{.DataContentType}}"}`,
	}

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		engine.Transform(ctx, event, transformation)
	}
}

// BenchmarkRateLimiter_Allow benchmarks rate limiter Allow() method
func BenchmarkRateLimiter_Allow(b *testing.B) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 10000,
		BurstSize:         100,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := ratelimiter.NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		limiter.Allow(ctx, "benchmark-key")
	}
}

// BenchmarkRateLimiter_Wait benchmarks rate limiter Wait() method
func BenchmarkRateLimiter_Wait(b *testing.B) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 10000,
		BurstSize:         100,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := ratelimiter.NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		limiter.Wait(ctx, "benchmark-key")
	}
}

// BenchmarkCircuitBreaker_Execute benchmarks circuit breaker execution
func BenchmarkCircuitBreaker_Execute(b *testing.B) {
	breaker := circuitbreaker.New(
		circuitbreaker.Config{
			FailureThreshold: 5,
			ResetTimeout:     1 * time.Second,
			HalfOpenMaxCalls: 3,
		},
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		breaker.Execute(func() error {
			return nil
		})
	}
}

// BenchmarkCircuitBreaker_State benchmarks state checking
func BenchmarkCircuitBreaker_State(b *testing.B) {
	breaker := circuitbreaker.New(
		circuitbreaker.Config{
			FailureThreshold: 5,
			ResetTimeout:     1 * time.Second,
			HalfOpenMaxCalls: 3,
		},
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		breaker.State()
	}
}

// BenchmarkCircuitBreaker_Reset benchmarks circuit breaker reset
func BenchmarkCircuitBreaker_Reset(b *testing.B) {
	breaker := circuitbreaker.New(
		circuitbreaker.Config{
			FailureThreshold: 5,
			ResetTimeout:     1 * time.Second,
			HalfOpenMaxCalls: 3,
		},
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		breaker.Reset()
	}
}

// BenchmarkRateLimiter_ConcurrentAccess benchmarks concurrent access to rate limiter
func BenchmarkRateLimiter_ConcurrentAccess(b *testing.B) {
	cfg := &config.RateLimitConfig{
		RequestsPerSecond: 100000,
		BurstSize:         1000,
		CleanupInterval:   1 * time.Hour,
		StaleThreshold:    24 * time.Hour,
	}

	limiter := ratelimiter.NewTokenBucketLimiter(cfg)
	defer limiter.Stop()

	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			limiter.Allow(ctx, "concurrent-key")
		}
	})
}
