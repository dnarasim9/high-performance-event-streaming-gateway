package kafka

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/config"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/logging"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/metrics"
)

// Producer implements the MessageBroker publish interface using Kafka.
type Producer struct {
	writer                   *kafka.Writer
	logger                   logging.Logger
	metrics                  metrics.MetricsCollector
	serializer               EventSerializer
	maxRetries               int
	retryBackoff             time.Duration
	circuitBreakerThreshold  int32
	failureCount             atomic.Int32
	successCount             atomic.Int32
	circuitOpen              bool
	mu                       sync.RWMutex
	circuitBreakerResetTimer *time.Timer
}

// NewProducer creates a new Producer.
func NewProducer(
	cfg *config.KafkaConfig,
	logger logging.Logger,
	metrics metrics.MetricsCollector,
) (*Producer, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Topic:                  cfg.Topic,
		WriteBackoffMin:        100 * time.Millisecond,
		WriteBackoffMax:        1 * time.Second,
		WriteTimeout:           cfg.WriteTimeout,
		ReadTimeout:            cfg.ReadTimeout,
		RequiredAcks:           kafka.RequireAll,
		AllowAutoTopicCreation: false,
		Compression:            kafka.Snappy,
	}

	producer := &Producer{
		writer:                  writer,
		logger:                  logger,
		metrics:                 metrics,
		maxRetries:              3,
		retryBackoff:            100 * time.Millisecond,
		circuitBreakerThreshold: 5,
		circuitOpen:             false,
	}

	return producer, nil
}

// Publish sends a single event to Kafka.
func (kp *Producer) Publish(ctx context.Context, event *entity.Event, partitionKey string) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	// Check circuit breaker
	kp.mu.RLock()
	if kp.circuitOpen {
		kp.mu.RUnlock()
		return fmt.Errorf("circuit breaker is open")
	}
	kp.mu.RUnlock()

	// Serialize the event
	serialized, err := kp.serializer.Serialize(event)
	if err != nil {
		kp.recordFailure()
		kp.logger.Error("failed to serialize event", logging.Field{Key: "error", Value: err})
		if kp.metrics != nil {
			kp.metrics.RecordError("publish", "serialization_error")
		}
		return fmt.Errorf("serialization failed: %w", err)
	}

	// Create Kafka message
	messages := []kafka.Message{
		{
			Key:   []byte(event.PartitionKey()),
			Value: serialized,
			Headers: []kafka.Header{
				{
					Key:   "event_id",
					Value: []byte(event.ID()),
				},
				{
					Key:   "timestamp",
					Value: []byte(event.Timestamp().String()),
				},
			},
		},
	}

	// Attempt publish with retry logic
	var lastErr error
	for attempt := 0; attempt <= kp.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(kp.retryBackoff * time.Duration(1<<uint(attempt))):
			}
		}

		startTime := time.Now()
		err := kp.writer.WriteMessages(ctx, messages...)
		duration := time.Since(startTime)

		if kp.metrics != nil {
			kp.metrics.RecordPublishDuration(duration)
		}

		if err == nil {
			kp.recordSuccess()
			if kp.metrics != nil {
				kp.metrics.RecordEventPublished()
			}
			kp.logger.Debug("event published", logging.Field{Key: "event_id", Value: event.ID()})
			return nil
		}

		lastErr = err
		kp.logger.Warn(
			"publish attempt failed",
			logging.Field{Key: "attempt", Value: attempt + 1},
			logging.Field{Key: "error", Value: err},
		)
	}

	kp.recordFailure()
	if kp.metrics != nil {
		kp.metrics.RecordError("publish", "max_retries_exceeded")
	}
	return fmt.Errorf("failed to publish after %d retries: %w", kp.maxRetries, lastErr)
}

// PublishBatch sends multiple events to Kafka in a batch.
func (kp *Producer) PublishBatch(ctx context.Context, events []*entity.Event, partitionKey string) error {
	if len(events) == 0 {
		return fmt.Errorf("events slice cannot be empty")
	}

	// Check circuit breaker
	kp.mu.RLock()
	if kp.circuitOpen {
		kp.mu.RUnlock()
		return fmt.Errorf("circuit breaker is open")
	}
	kp.mu.RUnlock()

	messages := kp.serializeEvents(events)

	if len(messages) == 0 {
		return fmt.Errorf("no valid events to publish")
	}

	return kp.publishWithRetry(ctx, messages)
}

// serializeEvents converts events to Kafka messages.
func (kp *Producer) serializeEvents(events []*entity.Event) []kafka.Message {
	messages := make([]kafka.Message, 0, len(events))

	for _, event := range events {
		if event == nil {
			continue
		}

		serialized, err := kp.serializer.Serialize(event)
		if err != nil {
			kp.logger.Warn(
				"failed to serialize event in batch",
				logging.Field{Key: "event_id", Value: event.ID()},
				logging.Field{Key: "error", Value: err},
			)
			continue
		}

		messages = append(messages, kafka.Message{
			Key:   []byte(event.PartitionKey()),
			Value: serialized,
			Headers: []kafka.Header{
				{
					Key:   "event_id",
					Value: []byte(event.ID()),
				},
				{
					Key:   "timestamp",
					Value: []byte(event.Timestamp().String()),
				},
			},
		})
	}
	return messages
}

// publishWithRetry attempts to publish messages with retry logic.
func (kp *Producer) publishWithRetry(ctx context.Context, messages []kafka.Message) error {
	var lastErr error
	for attempt := 0; attempt <= kp.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(kp.retryBackoff * time.Duration(1<<uint(attempt))):
			}
		}

		startTime := time.Now()
		err := kp.writer.WriteMessages(ctx, messages...)
		duration := time.Since(startTime)

		if kp.metrics != nil {
			kp.metrics.RecordPublishDuration(duration)
		}

		if err == nil {
			kp.recordSuccess()
			if kp.metrics != nil {
				for range messages {
					kp.metrics.RecordEventPublished()
				}
			}
			kp.logger.Debug("batch published", logging.Field{Key: "event_count", Value: len(messages)})
			return nil
		}

		lastErr = err
	}

	kp.recordFailure()
	if kp.metrics != nil {
		kp.metrics.RecordError("publish_batch", "max_retries_exceeded")
	}
	return fmt.Errorf("failed to publish batch after %d retries: %w", kp.maxRetries, lastErr)
}

// Close gracefully shuts down the producer.
func (kp *Producer) Close() error {
	if err := kp.writer.Close(); err != nil {
		kp.logger.Error("failed to close Kafka writer", logging.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to close Kafka writer: %w", err)
	}

	kp.mu.Lock()
	if kp.circuitBreakerResetTimer != nil {
		kp.circuitBreakerResetTimer.Stop()
	}
	kp.mu.Unlock()

	return nil
}

// SetSerializer sets the event serializer for the producer.
func (kp *Producer) SetSerializer(serializer EventSerializer) {
	kp.serializer = serializer
}

// recordSuccess records a successful message and updates circuit breaker.
func (kp *Producer) recordSuccess() {
	// Use atomic operations for counters to reduce lock contention
	kp.successCount.Add(1)
	kp.failureCount.Store(0)

	// Check circuit breaker status under lock
	kp.mu.RLock()
	shouldClose := kp.circuitOpen && kp.successCount.Load() >= kp.circuitBreakerThreshold
	kp.mu.RUnlock()

	if shouldClose {
		kp.mu.Lock()
		if kp.circuitOpen && kp.successCount.Load() >= kp.circuitBreakerThreshold {
			kp.circuitOpen = false
			kp.successCount.Store(0)
			kp.logger.Info("circuit breaker closed")
		}
		kp.mu.Unlock()
	}
}

// recordFailure records a failed message and updates circuit breaker.
func (kp *Producer) recordFailure() {
	// Use atomic operations for counters to reduce lock contention
	kp.failureCount.Add(1)
	kp.successCount.Store(0)

	// Check circuit breaker status under lock
	kp.mu.RLock()
	shouldOpen := !kp.circuitOpen && kp.failureCount.Load() >= kp.circuitBreakerThreshold
	kp.mu.RUnlock()

	if shouldOpen {
		kp.mu.Lock()
		if !kp.circuitOpen && kp.failureCount.Load() >= kp.circuitBreakerThreshold {
			kp.circuitOpen = true
			kp.failureCount.Store(0)
			kp.logger.Warn("circuit breaker opened")

			// Schedule circuit breaker reset after 30 seconds
			if kp.circuitBreakerResetTimer != nil {
				kp.circuitBreakerResetTimer.Stop()
			}
			kp.circuitBreakerResetTimer = time.AfterFunc(30*time.Second, func() {
				kp.mu.Lock()
				kp.circuitOpen = false
				kp.failureCount.Store(0)
				kp.successCount.Store(0)
				kp.mu.Unlock()
				kp.logger.Info("circuit breaker auto-reset")
			})
		}
		kp.mu.Unlock()
	}
}
