package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/repository"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/replay"
)

func BenchmarkEventCreation(b *testing.B) {
	b.ReportAllocs()

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

func BenchmarkEventCreation_WithMetadata(b *testing.B) {
	b.ReportAllocs()

	metadata := map[string]string{
		"env":    "production",
		"region": "us-east-1",
		"team":   "platform",
	}

	for i := 0; i < b.N; i++ {
		event, _ := entity.NewEvent(
			"event-id",
			"test-source",
			"test.event",
			"test/subject",
			[]byte(`{"key": "value"}`),
			"application/json",
		)
		event.WithMetadata(metadata)
	}
}

func BenchmarkEventCreation_WithPriority(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		event, _ := entity.NewEvent(
			"event-id",
			"test-source",
			"test.event",
			"test/subject",
			[]byte(`{"key": "value"}`),
			"application/json",
		)
		event.WithPriority(valueobject.PriorityHigh)
	}
}

func BenchmarkEventCreation_Chained(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		event, _ := entity.NewEvent(
			"event-id",
			"test-source",
			"test.event",
			"test/subject",
			[]byte(`{"key": "value"}`),
			"application/json",
		)
		event.
			WithPriority(valueobject.PriorityHigh).
			WithMetadata(map[string]string{"env": "prod"}).
			WithCorrelation("corr-id", "cause-id").
			WithPartitionKey("partition-1")
	}
}

func BenchmarkEventFilter_Matches(b *testing.B) {
	filter, _ := valueobject.NewEventFilter(
		"test.*",
		"event\\..*",
		".*subject",
		map[string]string{"env": "prod"},
		valueobject.PriorityMedium,
	)

	event, _ := entity.NewEvent(
		"event-id",
		"test-source",
		"event.created",
		"test/subject",
		[]byte(`{"key": "value"}`),
		"application/json",
	)
	event = event.WithMetadata(map[string]string{"env": "prod"}).WithPriority(valueobject.PriorityHigh)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Matches(event)
	}
}

func BenchmarkEventFilter_Matches_NoPattern(b *testing.B) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)

	event, _ := entity.NewEvent(
		"event-id",
		"test-source",
		"event.created",
		"test/subject",
		[]byte(`{"key": "value"}`),
		"application/json",
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Matches(event)
	}
}

func BenchmarkEventFilter_Matches_ComplexRegex(b *testing.B) {
	filter, _ := valueobject.NewEventFilter(
		`^(api|service)\.(prod|staging)\.(create|update|delete)$`,
		`^(user|order|payment)\.(created|updated|deleted|processed)$`,
		`^/(users|orders|payments)/[0-9a-f-]{36}/?$`,
		map[string]string{
			"env":      "production",
			"region":   "us-east-1",
			"team":     "backend",
			"priority": "high",
		},
		valueobject.PriorityCritical,
	)

	event, _ := entity.NewEvent(
		"550e8400-e29b-41d4-a716-446655440000",
		"api.prod.create",
		"user.created",
		"/users/550e8400-e29b-41d4-a716-446655440000",
		[]byte(`{"id": "550e8400-e29b-41d4-a716-446655440000", "name": "Test User"}`),
		"application/json",
	)
	event = event.
		WithMetadata(map[string]string{
			"env":      "production",
			"region":   "us-east-1",
			"team":     "backend",
			"priority": "high",
		}).
		WithPriority(valueobject.PriorityCritical)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Matches(event)
	}
}

func BenchmarkInMemoryEventStore_Store(b *testing.B) {
	store := replay.NewInMemoryEventStore(1 * time.Second) // Short TTL to auto-cleanup
	ctx := context.Background()

	event, _ := entity.NewEvent(
		"event-id",
		"test-source",
		"test.event",
		"test/subject",
		[]byte(`{"key": "value"}`),
		"application/json",
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store.Store(ctx, event)
	}
}

func BenchmarkInMemoryEventStore_Query(b *testing.B) {
	store := replay.NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	// Pre-populate store with events
	for i := 0; i < 1000; i++ {
		event, _ := entity.NewEvent(
			"event-id",
			"test-source",
			"test.event",
			"test/subject",
			[]byte(`{"key": "value"}`),
			"application/json",
		)
		store.Store(ctx, event)
	}

	filter, _ := valueobject.NewEventFilter("test.*", ".*", ".*", nil, valueobject.PriorityUnspecified)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store.Query(ctx, filter, nil)
	}
}

func BenchmarkInMemoryEventStore_Query_WithTimeRange(b *testing.B) {
	store := replay.NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	now := time.Now().UTC()

	// Pre-populate store with events
	for i := 0; i < 1000; i++ {
		event, _ := entity.NewEvent(
			"event-id",
			"test-source",
			"test.event",
			"test/subject",
			[]byte(`{"key": "value"}`),
			"application/json",
		)
		store.Store(ctx, event)
	}

	filter, _ := valueobject.NewEventFilter("test.*", ".*", ".*", nil, valueobject.PriorityUnspecified)
	timeRange := &repository.TimeRange{
		Start: now.Add(-5 * time.Minute),
		End:   now.Add(5 * time.Minute),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store.Query(ctx, filter, timeRange)
	}
}

func BenchmarkInMemoryEventStore_GetByID(b *testing.B) {
	store := replay.NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	// Store a single event
	event, _ := entity.NewEvent(
		"test-event-id",
		"test-source",
		"test.event",
		"test/subject",
		[]byte(`{"key": "value"}`),
		"application/json",
	)
	store.Store(ctx, event)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store.GetByID(ctx, "test-event-id")
	}
}

func BenchmarkInMemoryEventStore_GetLatest(b *testing.B) {
	store := replay.NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	// Pre-populate store with events
	for i := 0; i < 100; i++ {
		event, _ := entity.NewEvent(
			"event-id",
			"test-source",
			"test.event",
			"test/subject",
			[]byte(`{"key": "value"}`),
			"application/json",
		)
		store.Store(ctx, event)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		store.GetLatest(ctx, 10)
	}
}

func BenchmarkPriority_String(b *testing.B) {
	priorities := []valueobject.Priority{
		valueobject.PriorityUnspecified,
		valueobject.PriorityLow,
		valueobject.PriorityMedium,
		valueobject.PriorityHigh,
		valueobject.PriorityCritical,
	}

	b.ReportAllocs()
	b.ResetTimer()

	var s string
	for i := 0; i < b.N; i++ {
		for _, p := range priorities {
			s = p.String()
		}
	}
	_ = s
}

func BenchmarkEvent_IsHighPriority(b *testing.B) {
	event, _ := entity.NewEvent(
		"event-id",
		"test-source",
		"test.event",
		"test/subject",
		[]byte(`{"key": "value"}`),
		"application/json",
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event.IsHighPriority()
	}
}

func BenchmarkEvent_Data(b *testing.B) {
	event, _ := entity.NewEvent(
		"event-id",
		"test-source",
		"test.event",
		"test/subject",
		[]byte(`{"key": "value", "nested": {"field": "data"}}`),
		"application/json",
	)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event.Data()
	}
}

func BenchmarkEvent_Metadata(b *testing.B) {
	event, _ := entity.NewEvent(
		"event-id",
		"test-source",
		"test.event",
		"test/subject",
		[]byte(`{"key": "value"}`),
		"application/json",
	)
	event = event.WithMetadata(map[string]string{
		"env":    "prod",
		"region": "us-east-1",
		"team":   "platform",
	})

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event.Metadata()
	}
}

func BenchmarkConsumer_Creation(b *testing.B) {
	filter, _ := valueobject.NewEventFilter(".*", ".*", ".*", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		entity.NewConsumer("consumer-id", "consumer-name", "group-id", filter, transformation)
	}
}

func BenchmarkConsumer_StatusTransitions(b *testing.B) {
	filter, _ := valueobject.NewEventFilter(".*", ".*", ".*", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := entity.NewConsumer("consumer-id", "consumer-name", "group-id", filter, transformation)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		consumer.Activate()
		consumer.Deactivate()
	}
}

func BenchmarkNewEventFilter(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		valueobject.NewEventFilter(
			"source.*",
			"type\\..*",
			"subject.*",
			map[string]string{"env": "prod"},
			valueobject.PriorityMedium,
		)
	}
}

func BenchmarkNewEventFilter_ComplexRegex(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		valueobject.NewEventFilter(
			`^(api|service)\.(prod|staging)\.(create|update|delete)$`,
			`^(user|order|payment)\.(created|updated|deleted|processed)$`,
			`^/(users|orders|payments)/[0-9a-f-]{36}/?$`,
			map[string]string{
				"env":      "production",
				"region":   "us-east-1",
				"team":     "backend",
				"priority": "high",
			},
			valueobject.PriorityCritical,
		)
	}
}
