package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/port"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/repository"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// ReplayService implements the EventReplayUseCase interface.
// It retrieves and streams events from a specified time range.
type ReplayService struct {
	eventStore       port.EventStore
	metricsCollector port.MetricsCollector
	logger           *slog.Logger

	// Configuration
	batchSize         int
	rateLimitMS       int // Milliseconds between batches to rate limit replay
	channelBufferSize int
}

// NewReplayService creates a new ReplayService with the specified dependencies.
func NewReplayService(
	eventStore port.EventStore,
	metricsCollector port.MetricsCollector,
	logger *slog.Logger,
) *ReplayService {
	return &ReplayService{
		eventStore:        eventStore,
		metricsCollector:  metricsCollector,
		logger:            logger,
		batchSize:         100,
		rateLimitMS:       10,
		channelBufferSize: 100,
	}
}

// SetBatchSize sets the batch size for replay queries.
func (s *ReplayService) SetBatchSize(size int) {
	s.batchSize = size
}

// SetRateLimit sets the rate limit in milliseconds between batches.
func (s *ReplayService) SetRateLimit(ms int) {
	s.rateLimitMS = ms
}

// SetChannelBufferSize sets the buffer size for the event channel.
func (s *ReplayService) SetChannelBufferSize(size int) {
	s.channelBufferSize = size
}

// ReplayEvents retrieves and streams events from a specified time range.
// Optionally applies filters and limits the number of returned events.
// Returns a channel that receives replayed events or an error.
func (s *ReplayService) ReplayEvents(ctx context.Context, request *dto.ReplayRequest) (<-chan *entity.Event, error) {
	replayID := generateID()

	s.logger.InfoContext(
		ctx,
		"Starting event replay",
		slog.String("replayID", replayID),
		slog.Time("startTime", request.StartTime),
		slog.Time("endTime", request.EndTime),
		slog.Int("limit", request.Limit),
	)

	// Validate time range
	if request.StartTime.After(request.EndTime) {
		s.logger.ErrorContext(
			ctx,
			"Invalid time range for replay",
			slog.String("replayID", replayID),
			slog.Time("startTime", request.StartTime),
			slog.Time("endTime", request.EndTime),
		)
		s.metricsCollector.RecordError("ReplayEvents", "invalid_time_range")
		return nil, fmt.Errorf("start time cannot be after end time")
	}

	// Build filter from request
	var filter valueobject.EventFilter
	var err error

	if request.Filter != nil {
		filter, err = request.Filter.ToValueObject()
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to convert filter DTO to value object",
				slog.String("replayID", replayID),
				slog.Any("error", err),
			)
			s.metricsCollector.RecordError("ReplayEvents", "filter_conversion")
			return nil, fmt.Errorf("invalid filter: %w", err)
		}
	} else {
		// Empty filter matches all events
		f, err := valueobject.NewEventFilter("", "", "", nil, entity.PriorityUnspecified)
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to create default event filter",
				slog.String("replayID", replayID),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("invalid default filter: %w", err)
		}
		filter = f
	}

	// Create output channel
	outputChannel := make(chan *entity.Event, s.channelBufferSize)

	// Run replay in a goroutine to stream events asynchronously
	go s.streamReplayEvents(ctx, replayID, filter, request.StartTime, request.EndTime, request.Limit, outputChannel)

	return outputChannel, nil
}

// streamReplayEvents retrieves events from the event store and streams them to the output channel.
// It applies rate limiting to avoid overwhelming consumers.
func (s *ReplayService) streamReplayEvents(
	ctx context.Context,
	replayID string,
	filter valueobject.EventFilter,
	startTime, endTime time.Time,
	limit int,
	outputChannel chan<- *entity.Event,
) {
	defer close(outputChannel)

	startReplay := time.Now()
	eventCount := 0
	rateLimiter := time.NewTicker(time.Duration(s.rateLimitMS) * time.Millisecond)
	defer rateLimiter.Stop()

	timeRange := repository.NewTimeRange(startTime, endTime)

	// Query events from the store
	queryStart := time.Now()
	events, err := s.eventStore.Query(ctx, filter, timeRange)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to query events for replay",
			slog.String("replayID", replayID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordLatency("replay_query", float64(time.Since(queryStart).Milliseconds()))
		s.metricsCollector.RecordError("ReplayEvents", "query")
		return
	}
	s.metricsCollector.RecordLatency("replay_query", float64(time.Since(queryStart).Milliseconds()))

	s.logger.InfoContext(
		ctx,
		"Query completed for replay",
		slog.String("replayID", replayID),
		slog.Int("eventCount", len(events)),
		slog.Duration("duration", time.Since(queryStart)),
	)

	// Apply limit if specified
	if limit > 0 && len(events) > limit {
		events = events[:limit]
	}

	// Stream events with rate limiting
	for i, event := range events {
		// Check context for cancellation
		select {
		case <-ctx.Done():
			s.logger.InfoContext(
				ctx,
				"Replay canceled",
				slog.String("replayID", replayID),
				slog.Int("eventsStreamed", eventCount),
				slog.Duration("duration", time.Since(startReplay)),
			)
			return
		default:
		}

		// Try to send the event
		select {
		case <-ctx.Done():
			s.logger.InfoContext(
				ctx,
				"Replay canceled during send",
				slog.String("replayID", replayID),
				slog.Int("eventsStreamed", eventCount),
			)
			return

		case outputChannel <- event:
			eventCount++

			// Rate limiting: wait before sending the next batch
			if (i+1)%s.batchSize == 0 && i+1 < len(events) {
				<-rateLimiter.C
			}
		}
	}

	s.logger.InfoContext(
		ctx,
		"Replay completed successfully",
		slog.String("replayID", replayID),
		slog.Int("eventsStreamed", eventCount),
		slog.Duration("duration", time.Since(startReplay)),
	)

	// Record metrics
	s.metricsCollector.RecordLatency("replay_total", float64(time.Since(startReplay).Milliseconds()))
}
