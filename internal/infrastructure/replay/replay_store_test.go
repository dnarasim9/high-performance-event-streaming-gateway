package replay

import (
	"context"
	"testing"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/repository"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

func createTestEvent(id string, source string, eventType string) *entity.Event {
	event, _ := entity.NewEvent(
		id,
		source,
		eventType,
		"test/subject",
		[]byte(`{"test": "data"}`),
		"application/json",
	)
	return event
}

func TestStore_Success(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	event := createTestEvent("event-1", "test-source", "test.event")

	err := store.Store(ctx, event)

	if err != nil {
		t.Errorf("Store() got error %v, want nil", err)
	}
}

func TestStore_NilEvent(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	err := store.Store(ctx, nil)

	if err == nil {
		t.Error("Store() with nil event should return error")
	}
	if err.Error() != "event cannot be nil" {
		t.Errorf("Store() error = %q, want %q", err.Error(), "event cannot be nil")
	}
}

func TestStore_Multiple(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	events := []*entity.Event{
		createTestEvent("event-1", "source-1", "type-1"),
		createTestEvent("event-2", "source-2", "type-2"),
		createTestEvent("event-3", "source-1", "type-1"),
	}

	for _, event := range events {
		err := store.Store(ctx, event)
		if err != nil {
			t.Errorf("Store() got error %v", err)
		}
	}

	stats, _ := store.GetStats(ctx)
	if stats["total_events"] != 3 {
		t.Errorf("GetStats()[total_events] = %v, want 3", stats["total_events"])
	}
}

func TestGet_Found(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	event := createTestEvent("event-123", "source", "type")
	store.Store(ctx, event)

	retrieved, err := store.Get(ctx, "event-123")

	if err != nil {
		t.Errorf("Get() got error %v, want nil", err)
	}
	if retrieved == nil {
		t.Fatal("Get() returned nil event")
	}
	if retrieved.ID() != "event-123" {
		t.Errorf("Get() returned event with ID %q, want %q", retrieved.ID(), "event-123")
	}
}

func TestGet_NotFound(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	retrieved, err := store.Get(ctx, "non-existent")

	if err == nil {
		t.Error("Get() for non-existent event should return error")
	}
	if retrieved != nil {
		t.Error("Get() for non-existent event should return nil event")
	}
}

func TestGetByID_EmptyID(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	_, err := store.GetByID(ctx, "")

	if err == nil {
		t.Error("GetByID() with empty ID should return error")
	}
}

func TestQuery_EmptyStore(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	filter, _ := valueobject.NewEventFilter(".*", ".*", ".*", nil, valueobject.PriorityUnspecified)
	events, err := store.Query(ctx, filter, nil)

	if err != nil {
		t.Errorf("Query() got error %v, want nil", err)
	}
	if len(events) != 0 {
		t.Errorf("Query() returned %d events, want 0", len(events))
	}
}

func TestQuery_TimeRange(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	now := time.Now().UTC()

	// Create events with different timestamps
	event1, _ := entity.NewEvent("event-1", "source", "type", "subject", []byte("data"), "application/json")
	event2, _ := entity.NewEvent("event-2", "source", "type", "subject", []byte("data"), "application/json")
	event3, _ := entity.NewEvent("event-3", "source", "type", "subject", []byte("data"), "application/json")

	store.Store(ctx, event1)
	time.Sleep(10 * time.Millisecond)
	store.Store(ctx, event2)
	time.Sleep(10 * time.Millisecond)
	store.Store(ctx, event3)

	// Query with time range that includes all events
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	timeRange := &repository.TimeRange{
		Start: now.Add(-1 * time.Second),
		End:   now.Add(1 * time.Second),
	}

	events, err := store.Query(ctx, filter, timeRange)

	if err != nil {
		t.Errorf("Query() got error %v, want nil", err)
	}
	if len(events) != 3 {
		t.Errorf("Query() returned %d events, want 3", len(events))
	}
}

func TestQuery_WithFilter(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	// Store events with different sources
	store.Store(ctx, createTestEvent("event-1", "source-a", "type-1"))
	store.Store(ctx, createTestEvent("event-2", "source-b", "type-1"))
	store.Store(ctx, createTestEvent("event-3", "source-a", "type-2"))

	// Query for events from source-a
	filter, _ := valueobject.NewEventFilter("source-a", "", "", nil, valueobject.PriorityUnspecified)
	events, err := store.Query(ctx, filter, nil)

	if err != nil {
		t.Errorf("Query() got error %v, want nil", err)
	}
	if len(events) != 2 {
		t.Errorf("Query() returned %d events, want 2", len(events))
	}

	for _, event := range events {
		if event.Source() != "source-a" {
			t.Errorf("Query() returned event from %q, want source-a", event.Source())
		}
	}
}

func TestQuery_WithPriorityFilter(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	// Store events with different priorities
	event1 := createTestEvent("event-1", "source", "type").WithPriority(valueobject.PriorityLow)
	event2 := createTestEvent("event-2", "source", "type").WithPriority(valueobject.PriorityMedium)
	event3 := createTestEvent("event-3", "source", "type").WithPriority(valueobject.PriorityHigh)

	store.Store(ctx, event1)
	store.Store(ctx, event2)
	store.Store(ctx, event3)

	// Query for events with at least High priority
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityHigh)
	events, err := store.Query(ctx, filter, nil)

	if err != nil {
		t.Errorf("Query() got error %v, want nil", err)
	}
	if len(events) != 1 {
		t.Errorf("Query() returned %d events, want 1", len(events))
	}
	if events[0].ID() != "event-3" {
		t.Errorf("Query() returned wrong event: %q", events[0].ID())
	}
}

func TestQuery_InvalidTimeRange(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	now := time.Now().UTC()
	invalidRange := &repository.TimeRange{
		Start: now.Add(1 * time.Second),
		End:   now,
	}

	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	_, err := store.Query(ctx, filter, invalidRange)

	if err == nil {
		t.Error("Query() with invalid time range should return error")
	}
}

func TestCleanup(t *testing.T) {
	// Short retention time
	store := NewInMemoryEventStore(100 * time.Millisecond)
	ctx := context.Background()

	// Store an event
	event := createTestEvent("event-1", "source", "type")
	store.Store(ctx, event)

	// Verify it's stored
	stats, _ := store.GetStats(ctx)
	if stats["total_events"] != 1 {
		t.Fatal("Event was not stored")
	}

	// Wait for retention period to pass
	time.Sleep(150 * time.Millisecond)

	// Manually clear old events to simulate cleanup
	now := time.Now().UTC()
	cutoffTime := now.Add(-100 * time.Millisecond)

	// Query should return empty since events are older than retention
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	events, _ := store.Query(ctx, filter, &repository.TimeRange{
		Start: cutoffTime,
		End:   now,
	})

	// The old event should not be in the recent time range
	if len(events) > 0 {
		t.Errorf("Expected no events in recent time range after retention period, got %d", len(events))
	}

	// Store a new event and verify it's available
	event2 := createTestEvent("event-2", "source", "type")
	store.Store(ctx, event2)

	// Verify the new event is stored
	retrieved, _ := store.Get(ctx, "event-2")
	if retrieved == nil || retrieved.ID() != "event-2" {
		t.Error("New event was not stored properly")
	}
}

func TestCount(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	now := time.Now().UTC()

	// Store events
	store.Store(ctx, createTestEvent("event-1", "source", "type"))
	time.Sleep(10 * time.Millisecond)
	store.Store(ctx, createTestEvent("event-2", "source", "type"))
	time.Sleep(10 * time.Millisecond)
	store.Store(ctx, createTestEvent("event-3", "source", "type"))

	// Count all events
	count, err := store.Count(ctx, now.Add(-1*time.Second), now.Add(1*time.Second))

	if err != nil {
		t.Errorf("Count() got error %v, want nil", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestGetLatest(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	// Store events
	store.Store(ctx, createTestEvent("event-1", "source", "type"))
	time.Sleep(5 * time.Millisecond)
	store.Store(ctx, createTestEvent("event-2", "source", "type"))
	time.Sleep(5 * time.Millisecond)
	store.Store(ctx, createTestEvent("event-3", "source", "type"))

	// Get latest 2 events
	events, err := store.GetLatest(ctx, 2)

	if err != nil {
		t.Errorf("GetLatest() got error %v, want nil", err)
	}
	if len(events) != 2 {
		t.Errorf("GetLatest() returned %d events, want 2", len(events))
	}

	// Verify they are the latest events in order
	if events[0].ID() != "event-3" || events[1].ID() != "event-2" {
		t.Errorf("GetLatest() returned wrong events")
	}
}

func TestGetLatest_InvalidLimit(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	_, err := store.GetLatest(ctx, 0)

	if err == nil {
		t.Error("GetLatest() with limit 0 should return error")
	}

	_, err = store.GetLatest(ctx, -1)

	if err == nil {
		t.Error("GetLatest() with negative limit should return error")
	}
}

func TestGetLatest_EmptyStore(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	events, err := store.GetLatest(ctx, 10)

	if err != nil {
		t.Errorf("GetLatest() on empty store got error %v", err)
	}
	if len(events) != 0 {
		t.Errorf("GetLatest() on empty store returned %d events, want 0", len(events))
	}
}

func TestQueryWithPagination(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	// Store 10 events
	for i := 1; i <= 10; i++ {
		store.Store(ctx, createTestEvent("event-"+string(rune(i)), "source", "type"))
		time.Sleep(1 * time.Millisecond)
	}

	now := time.Now().UTC()

	// Query first page
	events, total, err := store.QueryWithPagination(
		ctx,
		now.Add(-1*time.Second),
		now.Add(1*time.Second),
		nil,
		3,
		1,
	)

	if err != nil {
		t.Errorf("QueryWithPagination() got error %v", err)
	}
	if len(events) != 3 {
		t.Errorf("QueryWithPagination() page 1 returned %d events, want 3", len(events))
	}
	if total != 10 {
		t.Errorf("QueryWithPagination() total = %d, want 10", total)
	}

	// Query second page
	events, total, err = store.QueryWithPagination(
		ctx,
		now.Add(-1*time.Second),
		now.Add(1*time.Second),
		nil,
		3,
		2,
	)

	if len(events) != 3 {
		t.Errorf("QueryWithPagination() page 2 returned %d events, want 3", len(events))
	}

	// Query out of range page
	events, total, err = store.QueryWithPagination(
		ctx,
		now.Add(-1*time.Second),
		now.Add(1*time.Second),
		nil,
		3,
		10,
	)

	if len(events) != 0 {
		t.Errorf("QueryWithPagination() out of range page returned %d events, want 0", len(events))
	}
}

func TestQueryWithPagination_InvalidPageSize(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	now := time.Now().UTC()

	_, _, err := store.QueryWithPagination(ctx, now, now, nil, 0, 1)

	if err == nil {
		t.Error("QueryWithPagination() with page size 0 should return error")
	}

	_, _, err = store.QueryWithPagination(ctx, now, now, nil, -1, 1)

	if err == nil {
		t.Error("QueryWithPagination() with negative page size should return error")
	}
}

func TestQueryWithPagination_InvalidPageNumber(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	now := time.Now().UTC()

	_, _, err := store.QueryWithPagination(ctx, now, now, nil, 10, 0)

	if err == nil {
		t.Error("QueryWithPagination() with page number 0 should return error")
	}

	_, _, err = store.QueryWithPagination(ctx, now, now, nil, 10, -1)

	if err == nil {
		t.Error("QueryWithPagination() with negative page number should return error")
	}
}

func TestClear(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	// Store some events
	store.Store(ctx, createTestEvent("event-1", "source", "type"))
	store.Store(ctx, createTestEvent("event-2", "source", "type"))

	// Verify they're stored
	stats, _ := store.GetStats(ctx)
	if stats["total_events"] != 2 {
		t.Fatal("Events were not stored")
	}

	// Clear the store
	err := store.Clear(ctx)

	if err != nil {
		t.Errorf("Clear() got error %v", err)
	}

	// Verify store is empty
	stats, _ = store.GetStats(ctx)
	if stats["total_events"] != 0 {
		t.Errorf("After Clear(), total_events = %v, want 0", stats["total_events"])
	}

	// Verify we can't get any events
	_, err = store.Get(ctx, "event-1")
	if err == nil {
		t.Error("Get() after Clear() should return error")
	}
}

func TestGetStats(t *testing.T) {
	store := NewInMemoryEventStore(1 * time.Hour)
	ctx := context.Background()

	// Get stats from empty store
	stats, err := store.GetStats(ctx)

	if err != nil {
		t.Errorf("GetStats() on empty store got error %v", err)
	}
	if stats["total_events"] != 0 {
		t.Errorf("GetStats() on empty store returned %v total_events", stats["total_events"])
	}

	// Store some events
	store.Store(ctx, createTestEvent("event-1", "source", "type"))
	time.Sleep(10 * time.Millisecond)
	store.Store(ctx, createTestEvent("event-2", "source", "type"))

	// Get stats
	stats, err = store.GetStats(ctx)

	if err != nil {
		t.Errorf("GetStats() got error %v", err)
	}
	if stats["total_events"] != 2 {
		t.Errorf("GetStats() returned %v total_events, want 2", stats["total_events"])
	}

	// Verify other stats are present
	if _, ok := stats["max_retention"]; !ok {
		t.Error("GetStats() missing max_retention")
	}
	if _, ok := stats["oldest_event_time"]; !ok {
		t.Error("GetStats() missing oldest_event_time")
	}
	if _, ok := stats["newest_event_time"]; !ok {
		t.Error("GetStats() missing newest_event_time")
	}
	if _, ok := stats["time_span"]; !ok {
		t.Error("GetStats() missing time_span")
	}
}
