package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/port"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// EventService implements the EventIngestionUseCase interface.
// It orchestrates event ingestion, validation, storage, and publishing.
type EventService struct {
	eventStore       port.EventStore
	eventPublisher   port.EventPublisher
	schemaValidator  port.SchemaValidator
	metricsCollector port.MetricsCollector
	logger           *slog.Logger
}

// NewEventService creates a new EventService with the specified dependencies.
func NewEventService(
	eventStore port.EventStore,
	eventPublisher port.EventPublisher,
	schemaValidator port.SchemaValidator,
	metricsCollector port.MetricsCollector,
	logger *slog.Logger,
) *EventService {
	return &EventService{
		eventStore:       eventStore,
		eventPublisher:   eventPublisher,
		schemaValidator:  schemaValidator,
		metricsCollector: metricsCollector,
		logger:           logger,
	}
}

// IngestEvent processes a single event for ingestion.
// It validates the event, validates against schema if specified, stores it, and publishes it.
// Returns the ID of the ingested event or an error.
func (s *EventService) IngestEvent(ctx context.Context, request *dto.IngestEventRequest) (string, error) {
	startTime := time.Now()
	eventID := generateID()
	correlationID := request.CorrelationID
	if correlationID == "" {
		correlationID = generateID()
	}

	s.logger.InfoContext(
		ctx,
		"Starting event ingestion",
		slog.String("correlationID", correlationID),
		slog.String("source", request.Source),
		slog.String("type", request.Type),
	)

	// Validate and enrich the event
	event, err := s.validateAndEnrichEvent(ctx, request, eventID, correlationID)
	if err != nil {
		return "", err
	}

	// Store and publish the event
	_, err = s.storeAndPublish(ctx, event, request.Type)
	if err != nil {
		return "", err
	}

	// Record metrics
	totalDuration := time.Since(startTime).Seconds()
	s.metricsCollector.RecordIngestion(1, totalDuration*1000)

	s.logger.InfoContext(
		ctx,
		"Event ingestion completed successfully",
		slog.String("correlationID", correlationID),
		slog.String("eventID", eventID),
		slog.Duration("duration", time.Since(startTime)),
	)

	return eventID, nil
}

// validateAndEnrichEvent validates the request, converts it to an entity, and enriches it with correlation data.
// Returns the enriched event or an error if validation fails.
func (s *EventService) validateAndEnrichEvent(ctx context.Context, request *dto.IngestEventRequest, eventID, correlationID string) (*entity.Event, error) {
	// Convert DTO to domain entity
	event, err := request.ToEntity(eventID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to convert request to event entity",
			slog.String("correlationID", correlationID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("IngestEvent", "validation")
		return nil, fmt.Errorf("invalid event request: %w", err)
	}

	// Apply correlation ID
	event = event.WithCorrelation(correlationID, request.CausationID)

	// Validate against schema if schema URL is provided
	if request.SchemaURL != "" {
		if err := s.validateSchema(ctx, request.SchemaURL, request.Data, correlationID); err != nil {
			s.metricsCollector.RecordError("IngestEvent", "schema_validation")
			return nil, err
		}
	}

	return event, nil
}

// validateSchema validates event data against a schema and records metrics.
func (s *EventService) validateSchema(ctx context.Context, schemaURL string, data []byte, correlationID string) error {
	schemaValidationStart := time.Now()
	if err := s.schemaValidator.Validate(ctx, schemaURL, data); err != nil {
		s.logger.ErrorContext(
			ctx,
			"Schema validation failed",
			slog.String("correlationID", correlationID),
			slog.String("schemaURL", schemaURL),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordLatency("schema_validation", float64(time.Since(schemaValidationStart).Milliseconds()))
		return fmt.Errorf("schema validation failed: %w", err)
	}
	s.metricsCollector.RecordLatency("schema_validation", float64(time.Since(schemaValidationStart).Milliseconds()))
	return nil
}

// storeAndPublish stores the event and publishes it to the message broker.
// Returns the event (or nil) and an error if the operation fails.
func (s *EventService) storeAndPublish(ctx context.Context, event *entity.Event, eventType string) (*entity.Event, error) {
	// Store event
	storeStart := time.Now()
	if err := s.eventStore.Store(ctx, event); err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to store event",
			slog.String("eventID", event.ID()),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordLatency("store", float64(time.Since(storeStart).Milliseconds()))
		s.metricsCollector.RecordError("IngestEvent", "storage")
		return nil, fmt.Errorf("failed to store event: %w", err)
	}
	s.metricsCollector.RecordLatency("store", float64(time.Since(storeStart).Milliseconds()))

	// Publish event
	publishStart := time.Now()
	topic := fmt.Sprintf("events.%s", eventType)
	if err := s.eventPublisher.Publish(ctx, event, topic); err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to publish event",
			slog.String("eventID", event.ID()),
			slog.String("topic", topic),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordLatency("publish", float64(time.Since(publishStart).Milliseconds()))
		s.metricsCollector.RecordError("IngestEvent", "publishing")
		return nil, fmt.Errorf("failed to publish event: %w", err)
	}
	s.metricsCollector.RecordLatency("publish", float64(time.Since(publishStart).Milliseconds()))

	return event, nil
}

// IngestEventBatch processes multiple events for ingestion.
// It validates and processes each event, collecting results and errors.
// Returns a slice of ingested event IDs or an error if the batch operation fails.
func (s *EventService) IngestEventBatch(ctx context.Context, requests []*dto.IngestEventRequest) ([]string, error) {
	startTime := time.Now()

	if len(requests) == 0 {
		return []string{}, nil
	}

	correlationID := requests[0].CorrelationID
	if correlationID == "" {
		correlationID = generateID()
	}

	s.logger.InfoContext(
		ctx,
		"Starting batch event ingestion",
		slog.String("correlationID", correlationID),
		slog.Int("eventCount", len(requests)),
	)

	// Prepare and validate all events
	events, eventIDs, err := s.prepareBatchEvents(ctx, requests, correlationID)
	if err != nil {
		return nil, err
	}

	// Store and publish all events
	if err := s.storeBatchEvents(ctx, events, eventIDs, correlationID); err != nil {
		return nil, err
	}

	if err := s.publishBatchEvents(ctx, events, correlationID); err != nil {
		return nil, err
	}

	// Record metrics
	totalDuration := time.Since(startTime).Seconds()
	s.metricsCollector.RecordIngestion(len(events), totalDuration*1000)

	s.logger.InfoContext(
		ctx,
		"Batch event ingestion completed successfully",
		slog.String("correlationID", correlationID),
		slog.Int("eventCount", len(events)),
		slog.Duration("duration", time.Since(startTime)),
	)

	return eventIDs, nil
}

// prepareBatchEvents validates all requests in a batch and converts them to entities.
// Returns the prepared events, their IDs, and an error if any request is invalid.
func (s *EventService) prepareBatchEvents(ctx context.Context, requests []*dto.IngestEventRequest, correlationID string) ([]*entity.Event, []string, error) {
	events := make([]*entity.Event, len(requests))
	eventIDs := make([]string, len(requests))

	for i, request := range requests {
		eventID := generateID()
		eventIDs[i] = eventID

		// Validate and enrich the event
		event, err := s.validateAndEnrichEvent(ctx, request, eventID, correlationID)
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to validate event in batch",
				slog.String("correlationID", correlationID),
				slog.Int("index", i),
				slog.Any("error", err),
			)
			s.metricsCollector.RecordError("IngestEventBatch", "validation")
			return nil, nil, fmt.Errorf("invalid event at index %d: %w", i, err)
		}

		events[i] = event
	}

	return events, eventIDs, nil
}

// storeBatchEvents stores all events in a batch to the event store.
func (s *EventService) storeBatchEvents(ctx context.Context, events []*entity.Event, eventIDs []string, correlationID string) error {
	storeStart := time.Now()
	for i, event := range events {
		if err := s.eventStore.Store(ctx, event); err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to store event in batch",
				slog.String("correlationID", correlationID),
				slog.Int("index", i),
				slog.String("eventID", eventIDs[i]),
				slog.Any("error", err),
			)
			s.metricsCollector.RecordLatency("batch_store", float64(time.Since(storeStart).Milliseconds()))
			s.metricsCollector.RecordError("IngestEventBatch", "storage")
			return fmt.Errorf("failed to store event at index %d: %w", i, err)
		}
	}
	s.metricsCollector.RecordLatency("batch_store", float64(time.Since(storeStart).Milliseconds()))
	return nil
}

// publishBatchEvents publishes all events in a batch to the message broker.
func (s *EventService) publishBatchEvents(ctx context.Context, events []*entity.Event, correlationID string) error {
	publishStart := time.Now()
	for i, event := range events {
		topic := fmt.Sprintf("events.%s", event.Type())
		if err := s.eventPublisher.Publish(ctx, event, topic); err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to publish event in batch",
				slog.String("correlationID", correlationID),
				slog.Int("index", i),
				slog.String("eventID", event.ID()),
				slog.String("topic", topic),
				slog.Any("error", err),
			)
			s.metricsCollector.RecordLatency("batch_publish", float64(time.Since(publishStart).Milliseconds()))
			s.metricsCollector.RecordError("IngestEventBatch", "publishing")
			return fmt.Errorf("failed to publish event at index %d: %w", i, err)
		}
	}
	s.metricsCollector.RecordLatency("batch_publish", float64(time.Since(publishStart).Milliseconds()))
	return nil
}

// generateID generates a unique ID using the current timestamp and a counter.
// This is a placeholder implementation and should be replaced with a proper ID generation strategy.
func generateID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%1000)
}
