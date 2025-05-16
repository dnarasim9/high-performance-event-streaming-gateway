package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/port"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	domainevent "github.com/dheemanth-hn/event-streaming-gateway/internal/domain/event"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// ConsumerService implements the ConsumerManagementUseCase interface.
// It manages consumer lifecycle including registration, deregistration, and querying.
type ConsumerService struct {
	consumerRepository port.ConsumerRepository
	metricsCollector   port.MetricsCollector
	logger             *slog.Logger
}

// NewConsumerService creates a new ConsumerService with the specified dependencies.
func NewConsumerService(
	consumerRepository port.ConsumerRepository,
	metricsCollector port.MetricsCollector,
	logger *slog.Logger,
) *ConsumerService {
	return &ConsumerService{
		consumerRepository: consumerRepository,
		metricsCollector:   metricsCollector,
		logger:             logger,
	}
}

// Register registers a new consumer in the system.
// Returns the ID of the registered consumer or an error.
func (s *ConsumerService) Register(ctx context.Context, request *dto.RegisterConsumerRequest) (string, error) {
	startTime := time.Now()
	consumerID := generateID()

	s.logger.InfoContext(
		ctx,
		"Registering new consumer",
		slog.String("consumerID", consumerID),
		slog.String("name", request.Name),
		slog.String("groupID", request.GroupID),
	)

	// Build filter from request
	var filter valueobject.EventFilter
	var err error

	if request.FilterConfig != nil {
		filter, err = request.FilterConfig.ToValueObject()
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to convert filter DTO to value object",
				slog.String("consumerID", consumerID),
				slog.Any("error", err),
			)
			s.metricsCollector.RecordError("Register", "filter_conversion")
			return "", fmt.Errorf("invalid filter configuration: %w", err)
		}
	} else {
		// Empty filter matches all events
		f, err := valueobject.NewEventFilter("", "", "", nil, entity.PriorityUnspecified)
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to create default event filter",
				slog.String("consumerID", consumerID),
				slog.Any("error", err),
			)
			return "", fmt.Errorf("invalid default filter: %w", err)
		}
		filter = f
	}

	// Build transformation from request
	var transformation valueobject.Transformation

	if request.TransformConfig != nil {
		transformation, err = request.TransformConfig.ToValueObject()
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to convert transformation DTO to value object",
				slog.String("consumerID", consumerID),
				slog.Any("error", err),
			)
			s.metricsCollector.RecordError("Register", "transformation_conversion")
			return "", fmt.Errorf("invalid transformation configuration: %w", err)
		}
	} else {
		// Default to no transformation
		t, err := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to create default transformation",
				slog.String("consumerID", consumerID),
				slog.Any("error", err),
			)
			return "", fmt.Errorf("invalid default transformation: %w", err)
		}
		transformation = t
	}

	// Create consumer entity
	consumer, err := entity.NewConsumer(
		consumerID,
		request.Name,
		request.GroupID,
		filter,
		transformation,
	)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to create consumer entity",
			slog.String("consumerID", consumerID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Register", "entity_creation")
		return "", fmt.Errorf("invalid consumer configuration: %w", err)
	}

	// Activate the consumer
	if err := consumer.Activate(); err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to activate consumer",
			slog.String("consumerID", consumerID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Register", "activation")
		return "", fmt.Errorf("failed to activate consumer: %w", err)
	}

	// Save consumer to repository
	if err := s.consumerRepository.Save(ctx, consumer); err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to save consumer to repository",
			slog.String("consumerID", consumerID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Register", "storage")
		return "", fmt.Errorf("failed to register consumer: %w", err)
	}

	s.logger.InfoContext(
		ctx,
		"Consumer registered successfully",
		slog.String("consumerID", consumerID),
		slog.String("name", request.Name),
		slog.String("groupID", request.GroupID),
		slog.Duration("duration", time.Since(startTime)),
	)

	// Emit domain event
	s.emitDomainEvent(ctx, domainevent.NewConsumerRegistered(consumer))

	// Record metrics
	s.metricsCollector.RecordLatency("register", float64(time.Since(startTime).Milliseconds()))

	return consumerID, nil
}

// Deregister removes a consumer from the system.
// Returns an error if the consumer cannot be deregistered.
func (s *ConsumerService) Deregister(ctx context.Context, consumerID string) error {
	startTime := time.Now()

	s.logger.InfoContext(
		ctx,
		"Deregistering consumer",
		slog.String("consumerID", consumerID),
	)

	// Retrieve the consumer
	consumer, err := s.consumerRepository.Get(ctx, consumerID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to retrieve consumer for deregistration",
			slog.String("consumerID", consumerID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Deregister", "retrieval")
		return fmt.Errorf("consumer not found: %w", err)
	}

	// Delete the consumer
	if err := s.consumerRepository.Delete(ctx, consumerID); err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to delete consumer from repository",
			slog.String("consumerID", consumerID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Deregister", "deletion")
		return fmt.Errorf("failed to deregister consumer: %w", err)
	}

	s.logger.InfoContext(
		ctx,
		"Consumer deregistered successfully",
		slog.String("consumerID", consumerID),
		slog.Duration("duration", time.Since(startTime)),
	)

	// Emit domain event
	s.emitDomainEvent(ctx, domainevent.NewConsumerDeregistered(consumer))

	// Record metrics
	s.metricsCollector.RecordLatency("deregister", float64(time.Since(startTime).Milliseconds()))

	return nil
}

// Get retrieves a consumer by its ID.
// Returns the consumer details or an error.
func (s *ConsumerService) Get(ctx context.Context, consumerID string) (*dto.ConsumerResponse, error) {
	s.logger.InfoContext(
		ctx,
		"Retrieving consumer",
		slog.String("consumerID", consumerID),
	)

	consumer, err := s.consumerRepository.Get(ctx, consumerID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to retrieve consumer",
			slog.String("consumerID", consumerID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Get", "retrieval")
		return nil, fmt.Errorf("consumer not found: %w", err)
	}

	return dto.FromEntityConsumer(consumer), nil
}

// List retrieves all consumers in a consumer group.
// Returns a slice of consumer details or an error.
func (s *ConsumerService) List(ctx context.Context, groupID string) ([]*dto.ConsumerResponse, error) {
	s.logger.InfoContext(
		ctx,
		"Listing consumers",
		slog.String("groupID", groupID),
	)

	consumers, err := s.consumerRepository.List(ctx, groupID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to list consumers",
			slog.String("groupID", groupID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("List", "retrieval")
		return nil, fmt.Errorf("failed to list consumers: %w", err)
	}

	return dto.FromEntityConsumers(consumers), nil
}

// emitDomainEvent emits a domain event for monitoring and auditing purposes.
// Currently a placeholder; should be integrated with an event bus.
func (s *ConsumerService) emitDomainEvent(ctx context.Context, domainEvent domainevent.DomainEvent) {
	s.logger.InfoContext(
		ctx,
		"Domain event emitted",
		slog.String("eventType", domainEvent.EventType()),
		slog.String("aggregateID", domainEvent.AggregateID()),
	)
}
