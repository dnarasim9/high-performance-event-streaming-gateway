package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/port"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// subscription represents an active subscription with its metadata.
type subscription struct {
	id           string
	consumerID   string
	groupID      string
	eventChannel <-chan *entity.Event
	cancel       context.CancelFunc
}

// SubscriptionService implements the EventSubscriptionUseCase interface.
// It manages active subscriptions and delivers events to subscribers.
type SubscriptionService struct {
	eventSubscriber  port.EventSubscriber
	metricsCollector port.MetricsCollector
	logger           *slog.Logger

	// Thread-safe storage for active subscriptions
	mu            sync.RWMutex
	subscriptions map[string]*subscription

	// Buffer size for event channels
	channelBufferSize int
}

// NewSubscriptionService creates a new SubscriptionService with the specified dependencies.
func NewSubscriptionService(
	eventSubscriber port.EventSubscriber,
	metricsCollector port.MetricsCollector,
	logger *slog.Logger,
) *SubscriptionService {
	return &SubscriptionService{
		eventSubscriber:   eventSubscriber,
		metricsCollector:  metricsCollector,
		logger:            logger,
		subscriptions:     make(map[string]*subscription),
		channelBufferSize: 1000,
	}
}

// SetChannelBufferSize sets the buffer size for event channels.
// This should be called before creating subscriptions.
func (s *SubscriptionService) SetChannelBufferSize(size int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.channelBufferSize = size
}

// Subscribe creates a new subscription to events matching the specified filter.
// Returns a channel that receives matching events or an error.
func (s *SubscriptionService) Subscribe(ctx context.Context, request *dto.SubscribeRequest) (<-chan *entity.Event, error) {
	subscriptionID := generateID()

	s.logger.InfoContext(
		ctx,
		"Creating subscription",
		slog.String("subscriptionID", subscriptionID),
		slog.String("groupID", request.GroupID),
		slog.String("consumerID", request.ConsumerID),
	)

	// Build filter from request
	var filter valueobject.EventFilter
	var err error

	if request.Filter != nil {
		filter, err = request.Filter.ToValueObject()
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to convert filter DTO to value object",
				slog.String("subscriptionID", subscriptionID),
				slog.Any("error", err),
			)
			s.metricsCollector.RecordError("Subscribe", "filter_conversion")
			return nil, fmt.Errorf("invalid filter: %w", err)
		}
	} else {
		// Empty filter matches all events
		f, err := valueobject.NewEventFilter("", "", "", nil, entity.PriorityUnspecified)
		if err != nil {
			s.logger.ErrorContext(
				ctx,
				"Failed to create default event filter",
				slog.String("subscriptionID", subscriptionID),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("invalid default filter: %w", err)
		}
		filter = f
	}

	// Create a topic based on group ID
	topic := fmt.Sprintf("consumer.%s", request.GroupID)

	// Subscribe to the message broker
	eventChannel, err := s.eventSubscriber.Subscribe(ctx, topic, request.GroupID, filter)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to subscribe to message broker",
			slog.String("subscriptionID", subscriptionID),
			slog.String("topic", topic),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Subscribe", "broker_subscription")
		return nil, fmt.Errorf("failed to subscribe to message broker: %w", err)
	}

	// Create a cancel context for this subscription
	cancelCtx, cancel := context.WithCancel(context.Background())

	// Store the subscription
	sub := &subscription{
		id:           subscriptionID,
		consumerID:   request.ConsumerID,
		groupID:      request.GroupID,
		eventChannel: eventChannel,
		cancel:       cancel,
	}

	s.mu.Lock()
	s.subscriptions[subscriptionID] = sub
	s.mu.Unlock()

	s.logger.InfoContext(
		ctx,
		"Subscription created successfully",
		slog.String("subscriptionID", subscriptionID),
		slog.String("consumerID", request.ConsumerID),
	)

	// Record metrics
	s.metricsCollector.RecordSubscription(request.ConsumerID, request.GroupID)

	// Create a wrapped channel that handles context cancellation
	outputChannel := make(chan *entity.Event, s.channelBufferSize)
	go s.forwardEvents(cancelCtx, subscriptionID, eventChannel, outputChannel)

	return outputChannel, nil
}

// Unsubscribe cancels an active subscription.
// Returns an error if the subscription cannot be canceled.
func (s *SubscriptionService) Unsubscribe(ctx context.Context, subscriptionID string) error {
	s.mu.Lock()
	sub, exists := s.subscriptions[subscriptionID]
	if !exists {
		s.mu.Unlock()
		s.logger.WarnContext(
			ctx,
			"Attempted to unsubscribe from non-existent subscription",
			slog.String("subscriptionID", subscriptionID),
		)
		return fmt.Errorf("subscription not found: %s", subscriptionID)
	}
	delete(s.subscriptions, subscriptionID)
	s.mu.Unlock()

	s.logger.InfoContext(
		ctx,
		"Canceling subscription",
		slog.String("subscriptionID", subscriptionID),
		slog.String("consumerID", sub.consumerID),
	)

	// Cancel the subscription context
	sub.cancel()

	// Unsubscribe from the message broker
	if err := s.eventSubscriber.Unsubscribe(); err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to unsubscribe from message broker",
			slog.String("subscriptionID", subscriptionID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Unsubscribe", "broker_unsubscription")
		return fmt.Errorf("failed to unsubscribe from message broker: %w", err)
	}

	s.logger.InfoContext(
		ctx,
		"Subscription canceled successfully",
		slog.String("subscriptionID", subscriptionID),
	)

	return nil
}

// forwardEvents forwards events from the broker channel to the output channel.
// It respects the subscription context for graceful shutdown.
func (s *SubscriptionService) forwardEvents(ctx context.Context, subscriptionID string, in <-chan *entity.Event, out chan<- *entity.Event) {
	defer close(out)

	for {
		select {
		case <-ctx.Done():
			s.logger.InfoContext(
				context.Background(),
				"Event forwarding stopped due to context cancellation",
				slog.String("subscriptionID", subscriptionID),
			)
			return

		case event, ok := <-in:
			if !ok {
				s.logger.InfoContext(
					context.Background(),
					"Event channel closed",
					slog.String("subscriptionID", subscriptionID),
				)
				return
			}

			// Try to send the event with a timeout to avoid blocking indefinitely
			select {
			case out <- event:
				// Event sent successfully
			case <-ctx.Done():
				s.logger.InfoContext(
					context.Background(),
					"Event forwarding stopped during send",
					slog.String("subscriptionID", subscriptionID),
				)
				return
			}
		}
	}
}

// GetActiveSubscriptions returns the count of active subscriptions.
// Useful for monitoring and debugging.
func (s *SubscriptionService) GetActiveSubscriptions() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.subscriptions)
}
