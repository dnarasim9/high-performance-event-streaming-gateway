package replay

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/repository"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// EventFilter is a function that determines if an event matches the filter criteria.
type EventFilter func(*entity.Event) bool

// InMemoryEventStore is an in-memory implementation of EventStore for replay functionality.
// TODO: Replace with persistent storage (e.g., TimescaleDB, ClickHouse) for production use.
type InMemoryEventStore struct {
	events       []*entity.Event
	maxRetention time.Duration
	lastCleanup  time.Time
	mu           sync.RWMutex
	// timeBuckets maps minute timestamp (Unix/60) to events for fast time-range queries
	timeBuckets map[int64][]*entity.Event
}

// NewInMemoryEventStore creates a new in-memory event store.
func NewInMemoryEventStore(maxRetention time.Duration) *InMemoryEventStore {
	store := &InMemoryEventStore{
		events:       make([]*entity.Event, 0),
		maxRetention: maxRetention,
		lastCleanup:  time.Now(),
		timeBuckets:  make(map[int64][]*entity.Event),
	}

	return store
}

// Store adds an event to the store and indexes it by time bucket.
func (ies *InMemoryEventStore) Store(ctx context.Context, event *entity.Event) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	ies.mu.Lock()
	defer ies.mu.Unlock()

	ies.events = append(ies.events, event)

	// Index event by minute bucket for fast time-range queries
	bucket := event.Timestamp().Unix() / 60
	ies.timeBuckets[bucket] = append(ies.timeBuckets[bucket], event)

	// Cleanup if necessary
	if time.Since(ies.lastCleanup) > 1*time.Minute {
		ies.cleanup()
		ies.lastCleanup = time.Now()
	}

	return nil
}

// Query implements repository.EventRepository.
func (ies *InMemoryEventStore) Query(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
	var start, end time.Time
	if timeRange != nil {
		start = timeRange.Start
		end = timeRange.End
	} else {
		start = time.Time{}
		end = time.Now()
	}
	return ies.queryByTime(ctx, start, end, func(event *entity.Event) bool {
		if filter.Empty() {
			return true
		}
		return filter.Matches(event)
	})
}

// queryByTime retrieves events from the store within a time range and optionally filtered.
// Uses time-bucket indexing to only scan relevant buckets instead of all events.
func (ies *InMemoryEventStore) queryByTime(
	_ctx context.Context,
	startTime, endTime time.Time,
	filter EventFilter,
) ([]*entity.Event, error) {
	if startTime.After(endTime) {
		return nil, fmt.Errorf("start time cannot be after end time")
	}

	ies.mu.RLock()
	defer ies.mu.RUnlock()

	result := make([]*entity.Event, 0)

	// Calculate minute buckets to scan
	startBucket := startTime.Unix() / 60
	endBucket := endTime.Unix() / 60

	// Scan only relevant time buckets
	for bucket := startBucket; bucket <= endBucket; bucket++ {
		if events, exists := ies.timeBuckets[bucket]; exists {
			for _, event := range events {
				// Check time range
				if event.Timestamp().Before(startTime) || event.Timestamp().After(endTime) {
					continue
				}

				// Apply filter if provided
				if filter != nil && !filter(event) {
					continue
				}

				result = append(result, event)
			}
		}
	}

	return result, nil
}

// QueryWithPagination retrieves events with pagination support.
func (ies *InMemoryEventStore) QueryWithPagination(
	ctx context.Context,
	startTime, endTime time.Time,
	filter EventFilter,
	pageSize int,
	pageNumber int,
) ([]*entity.Event, int, error) {
	if pageSize < 1 {
		return nil, 0, fmt.Errorf("page size must be at least 1")
	}

	if pageNumber < 1 {
		return nil, 0, fmt.Errorf("page number must be at least 1")
	}

	// Get all matching events
	allEvents, err := ies.queryByTime(ctx, startTime, endTime, filter)
	if err != nil {
		return nil, 0, err
	}

	// Calculate pagination
	totalCount := len(allEvents)
	startIdx := (pageNumber - 1) * pageSize
	endIdx := startIdx + pageSize

	if startIdx >= totalCount {
		return []*entity.Event{}, totalCount, nil
	}

	if endIdx > totalCount {
		endIdx = totalCount
	}

	return allEvents[startIdx:endIdx], totalCount, nil
}

// Count returns the number of events in the store within a time range.
func (ies *InMemoryEventStore) Count(ctx context.Context, startTime, endTime time.Time) (int, error) {
	ies.mu.RLock()
	defer ies.mu.RUnlock()

	count := 0
	for _, event := range ies.events {
		if event.Timestamp().After(startTime) && event.Timestamp().Before(endTime) {
			count++
		}
	}

	return count, nil
}

// GetByID retrieves a specific event by ID.
func (ies *InMemoryEventStore) GetByID(ctx context.Context, eventID string) (*entity.Event, error) {
	if eventID == "" {
		return nil, fmt.Errorf("event ID cannot be empty")
	}

	ies.mu.RLock()
	defer ies.mu.RUnlock()

	for _, event := range ies.events {
		if event.ID() == eventID {
			return event, nil
		}
	}

	return nil, fmt.Errorf("event %q not found", eventID)
}

// GetLatest returns the most recent N events.
func (ies *InMemoryEventStore) GetLatest(ctx context.Context, limit int) ([]*entity.Event, error) {
	if limit < 1 {
		return nil, fmt.Errorf("limit must be at least 1")
	}

	ies.mu.RLock()
	defer ies.mu.RUnlock()

	if len(ies.events) == 0 {
		return []*entity.Event{}, nil
	}

	// Events should already be sorted by timestamp, but ensure they are
	sorted := make([]*entity.Event, len(ies.events))
	copy(sorted, ies.events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp().After(sorted[j].Timestamp())
	})

	if limit > len(sorted) {
		limit = len(sorted)
	}

	return sorted[:limit], nil
}

// cleanup removes events that are older than maxRetention and updates time buckets.
func (ies *InMemoryEventStore) cleanup() {
	cutoffTime := time.Now().Add(-ies.maxRetention)
	newEvents := make([]*entity.Event, 0)

	// Clean up events list
	for _, event := range ies.events {
		if event.Timestamp().After(cutoffTime) {
			newEvents = append(newEvents, event)
		}
	}

	ies.events = newEvents

	// Clean up time buckets
	cutoffBucket := cutoffTime.Unix() / 60
	for bucket := range ies.timeBuckets {
		if bucket < cutoffBucket {
			delete(ies.timeBuckets, bucket)
		}
	}
}

// Clear removes all events from the store and clears time buckets.
func (ies *InMemoryEventStore) Clear(ctx context.Context) error {
	ies.mu.Lock()
	defer ies.mu.Unlock()

	ies.events = make([]*entity.Event, 0)
	ies.timeBuckets = make(map[int64][]*entity.Event)
	return nil
}

// Get retrieves an event by its ID (implements repository.EventRepository).
func (ies *InMemoryEventStore) Get(ctx context.Context, id string) (*entity.Event, error) {
	return ies.GetByID(ctx, id)
}

// GetStats returns statistics about the store.
func (ies *InMemoryEventStore) GetStats(ctx context.Context) (map[string]interface{}, error) {
	ies.mu.RLock()
	defer ies.mu.RUnlock()

	stats := map[string]interface{}{
		"total_events":  len(ies.events),
		"max_retention": ies.maxRetention.String(),
	}

	if len(ies.events) > 0 {
		// Find oldest and newest events
		oldest := ies.events[0]
		newest := ies.events[0]

		for _, event := range ies.events {
			if event.Timestamp().Before(oldest.Timestamp()) {
				oldest = event
			}
			if event.Timestamp().After(newest.Timestamp()) {
				newest = event
			}
		}

		stats["oldest_event_time"] = oldest.Timestamp()
		stats["newest_event_time"] = newest.Timestamp()
		stats["time_span"] = newest.Timestamp().Sub(oldest.Timestamp()).String()
	}

	return stats, nil
}
