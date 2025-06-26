package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/repository"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// Note: MockEventStore is already defined in event_service_test.go

// TestReplayService_ReplayEvents_Success tests successful event replay
func TestReplayService_ReplayEvents_Success(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	events := make([]*entity.Event, 2)
	for i := 0; i < 2; i++ {
		event, _ := entity.NewEvent(
			"event-"+string(rune('0'+i)),
			"test-source",
			"test.type",
			"/test",
			[]byte(`{"test": "data"}`),
			"application/json",
		)
		events[i] = event
	}

	store := &MockEventStore{
		QueryFunc: func(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
			return events, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	req := &dto.ReplayRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     0,
	}

	ch, err := service.ReplayEvents(context.Background(), req)
	if err != nil {
		t.Errorf("ReplayEvents() error = %v, want nil", err)
	}
	if ch == nil {
		t.Error("ReplayEvents() returned nil channel")
	}

	// Consume the channel to verify events are streamed
	eventCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for {
		select {
		case event := <-ch:
			if event != nil {
				eventCount++
			}
		case <-ctx.Done():
			goto done
		}
	}
done:
	if eventCount != 2 {
		t.Errorf("ReplayEvents() streamed %d events, want 2", eventCount)
	}
}

// TestReplayService_ReplayEvents_WithFilter tests replay with event filter
func TestReplayService_ReplayEvents_WithFilter(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	events := make([]*entity.Event, 1)
	event, _ := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"test": "data"}`),
		"application/json",
	)
	events[0] = event

	store := &MockEventStore{
		QueryFunc: func(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
			return events, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	req := &dto.ReplayRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     0,
		Filter: &dto.EventFilterDTO{
			TypePattern: "test.type",
		},
	}

	ch, err := service.ReplayEvents(context.Background(), req)
	if err != nil {
		t.Errorf("ReplayEvents() error = %v, want nil", err)
	}
	if ch == nil {
		t.Error("ReplayEvents() returned nil channel")
	}

	// Consume the channel
	eventCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for {
		select {
		case event := <-ch:
			if event != nil {
				eventCount++
			}
		case <-ctx.Done():
			goto done
		}
	}
done:
	if eventCount != 1 {
		t.Errorf("ReplayEvents() with filter streamed %d events, want 1", eventCount)
	}
}

// TestReplayService_ReplayEvents_WithLimit tests replay with event limit
func TestReplayService_ReplayEvents_WithLimit(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	events := make([]*entity.Event, 5)
	for i := 0; i < 5; i++ {
		event, _ := entity.NewEvent(
			"event-"+string(rune('0'+i)),
			"test-source",
			"test.type",
			"/test",
			[]byte(`{"test": "data"}`),
			"application/json",
		)
		events[i] = event
	}

	store := &MockEventStore{
		QueryFunc: func(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
			return events, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	req := &dto.ReplayRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     2,
	}

	ch, err := service.ReplayEvents(context.Background(), req)
	if err != nil {
		t.Errorf("ReplayEvents() error = %v, want nil", err)
	}

	// Consume the channel
	eventCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for {
		select {
		case event := <-ch:
			if event != nil {
				eventCount++
			}
		case <-ctx.Done():
			goto done
		}
	}
done:
	if eventCount != 2 {
		t.Errorf("ReplayEvents() with limit 2 streamed %d events, want 2", eventCount)
	}
}

// TestReplayService_ReplayEvents_InvalidTimeRange tests replay with invalid time range
func TestReplayService_ReplayEvents_InvalidTimeRange(t *testing.T) {
	endTime := time.Now()
	startTime := endTime.Add(1 * time.Hour)

	store := &MockEventStore{}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	req := &dto.ReplayRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     0,
	}

	_, err := service.ReplayEvents(context.Background(), req)
	if err == nil {
		t.Error("ReplayEvents() with invalid time range expected error, got nil")
	}
}

// TestReplayService_ReplayEvents_QueryError tests replay with query error
func TestReplayService_ReplayEvents_QueryError(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	store := &MockEventStore{
		QueryFunc: func(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
			return nil, errors.New("query error")
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	req := &dto.ReplayRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     0,
	}

	ch, err := service.ReplayEvents(context.Background(), req)
	if err != nil {
		t.Errorf("ReplayEvents() error = %v, want nil", err)
	}
	if ch == nil {
		t.Error("ReplayEvents() returned nil channel")
	}

	// Consume the channel to allow goroutine to complete
	for range ch {
	}
}

// TestReplayService_ReplayEvents_EmptyResult tests replay with empty result
func TestReplayService_ReplayEvents_EmptyResult(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	store := &MockEventStore{
		QueryFunc: func(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
			return []*entity.Event{}, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	req := &dto.ReplayRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     0,
	}

	ch, err := service.ReplayEvents(context.Background(), req)
	if err != nil {
		t.Errorf("ReplayEvents() error = %v, want nil", err)
	}

	// Consume the channel
	eventCount := 0
	for range ch {
		eventCount++
	}

	if eventCount != 0 {
		t.Errorf("ReplayEvents() with empty result streamed %d events, want 0", eventCount)
	}
}

// TestReplayService_SetBatchSize tests batch size configuration
func TestReplayService_SetBatchSize(t *testing.T) {
	store := &MockEventStore{}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	service.SetBatchSize(50)
	if service.batchSize != 50 {
		t.Errorf("SetBatchSize() = %d, want 50", service.batchSize)
	}
}

// TestReplayService_SetRateLimit tests rate limit configuration
func TestReplayService_SetRateLimit(t *testing.T) {
	store := &MockEventStore{}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	service.SetRateLimit(20)
	if service.rateLimitMS != 20 {
		t.Errorf("SetRateLimit() = %d, want 20", service.rateLimitMS)
	}
}

// TestReplayService_SetChannelBufferSize tests channel buffer size configuration
func TestReplayService_SetChannelBufferSize(t *testing.T) {
	store := &MockEventStore{}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	service.SetChannelBufferSize(500)
	if service.channelBufferSize != 500 {
		t.Errorf("SetChannelBufferSize() = %d, want 500", service.channelBufferSize)
	}
}

// TestReplayService_ReplayEvents_ContextCancellation tests replay with context cancellation
func TestReplayService_ReplayEvents_ContextCancellation(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	events := make([]*entity.Event, 100)
	for i := 0; i < 100; i++ {
		event, _ := entity.NewEvent(
			"event-"+string(rune('0'+(i/10))),
			"test-source",
			"test.type",
			"/test",
			[]byte(`{"test": "data"}`),
			"application/json",
		)
		events[i] = event
	}

	store := &MockEventStore{
		QueryFunc: func(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
			return events, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	ctx, cancel := context.WithCancel(context.Background())
	req := &dto.ReplayRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     0,
	}

	ch, err := service.ReplayEvents(ctx, req)
	if err != nil {
		t.Errorf("ReplayEvents() error = %v, want nil", err)
	}

	// Consume a few events then cancel
	<-ch
	<-ch
	cancel()

	// Consume remaining events
	for range ch {
	}
}

// TestReplayService_ReplayEvents_LargeEventSet tests replay with many events
func TestReplayService_ReplayEvents_LargeEventSet(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	// Create a large set of events
	events := make([]*entity.Event, 1000)
	for i := 0; i < 1000; i++ {
		event, _ := entity.NewEvent(
			"event-"+string(rune('0'+(i/100))),
			"test-source",
			"test.type",
			"/test",
			[]byte(`{"test": "data"}`),
			"application/json",
		)
		events[i] = event
	}

	store := &MockEventStore{
		QueryFunc: func(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
			return events, nil
		},
	}
	metrics := &MockMetricsCollector{}
	service := NewReplayService(store, metrics, newTestLogger())

	req := &dto.ReplayRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     500, // Apply limit
	}

	ch, err := service.ReplayEvents(context.Background(), req)
	if err != nil {
		t.Errorf("ReplayEvents() error = %v, want nil", err)
	}

	// Consume the channel
	eventCount := 0
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for {
		select {
		case event := <-ch:
			if event != nil {
				eventCount++
			}
		case <-ctx.Done():
			goto done
		}
	}
done:
	if eventCount != 500 {
		t.Errorf("ReplayEvents() with limit 500 streamed %d events, want 500", eventCount)
	}
}

// TestReplayService_ReplayEvents_MetricsRecorded tests that metrics are recorded
func TestReplayService_ReplayEvents_MetricsRecorded(t *testing.T) {
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	events := make([]*entity.Event, 1)
	event, _ := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"test": "data"}`),
		"application/json",
	)
	events[0] = event

	store := &MockEventStore{
		QueryFunc: func(ctx context.Context, filter valueobject.EventFilter, timeRange *repository.TimeRange) ([]*entity.Event, error) {
			return events, nil
		},
	}

	metricsRecorded := false
	metrics := &MockMetricsCollector{
		RecordLatencyFunc: func(operation string, durationMS float64) {
			if operation == "replay_query" || operation == "replay_total" {
				metricsRecorded = true
			}
		},
	}
	service := NewReplayService(store, metrics, newTestLogger())

	req := &dto.ReplayRequest{
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     0,
	}

	ch, err := service.ReplayEvents(context.Background(), req)
	if err != nil {
		t.Errorf("ReplayEvents() error = %v, want nil", err)
	}

	// Consume the channel to allow metrics to be recorded
	for range ch {
	}

	if !metricsRecorded {
		t.Error("ReplayEvents() did not record metrics")
	}
}
