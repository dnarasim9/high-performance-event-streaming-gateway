package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/config"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/logging"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/infrastructure/metrics"
)

// EventFilter is a function that determines if an event should be processed.
type EventFilter func(*entity.Event) bool

// Consumer implements the MessageBroker subscribe interface using Kafka.
type Consumer struct {
	reader        *kafka.Reader
	logger        logging.Logger
	metrics       metrics.MetricsCollector
	serializer    EventSerializer
	filter        EventFilter
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	isRunning     bool
	offsetManager *OffsetManager
}

// OffsetManager manages Kafka consumer offsets.
type OffsetManager struct {
	offsets map[int]int64
	mu      sync.RWMutex
}

// NewConsumer creates a new Consumer.
func NewConsumer(
	cfg *config.KafkaConfig,
	logger logging.Logger,
	metrics metrics.MetricsCollector,
) (*Consumer, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:                cfg.Brokers,
		Topic:                  cfg.Topic,
		GroupID:                cfg.GroupID,
		StartOffset:            kafka.LastOffset,
		CommitInterval:         cfg.CommitInterval,
		SessionTimeout:         cfg.SessionTimeout,
		MaxBytes:               10e6, // 10MB
		QueueCapacity:          100,
		PartitionWatchInterval: 5 * time.Second,
	})

	consumer := &Consumer{
		reader:        reader,
		logger:        logger,
		metrics:       metrics,
		isRunning:     false,
		offsetManager: &OffsetManager{offsets: make(map[int]int64)},
	}

	return consumer, nil
}

// Subscribe starts consuming messages from the configured topic.
func (kc *Consumer) Subscribe(ctx context.Context, eventChan chan<- *entity.Event, errorChan chan<- error) error {
	kc.mu.Lock()
	if kc.isRunning {
		kc.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}

	kc.ctx, kc.cancel = context.WithCancel(ctx)
	kc.isRunning = true
	kc.mu.Unlock()

	kc.wg.Add(1)
	go kc.consumeMessages(eventChan, errorChan)

	return nil
}

// consumeMessages runs the message consumption loop.
func (kc *Consumer) consumeMessages(eventChan chan<- *entity.Event, errorChan chan<- error) {
	defer kc.wg.Done()
	defer func() {
		kc.mu.Lock()
		kc.isRunning = false
		kc.mu.Unlock()
	}()

	for {
		select {
		case <-kc.ctx.Done():
			kc.logger.Info("consumer context canceled")
			return
		default:
		}

		kc.processMessage(eventChan, errorChan)
	}
}

// processMessage handles a single message from Kafka.
func (kc *Consumer) processMessage(eventChan chan<- *entity.Event, errorChan chan<- error) {
	msg, err := kc.reader.FetchMessage(kc.ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		kc.handleFetchError(err, errorChan)
		return
	}

	event, err := kc.serializer.Deserialize(msg.Value)
	if err != nil {
		kc.handleDeserializationError(msg)
		return
	}

	if kc.filter != nil && !kc.filter(event) {
		kc.commitMessage(msg)
		return
	}

	kc.offsetManager.UpdateOffset(msg.Partition, msg.Offset)
	if kc.metrics != nil {
		kc.metrics.RecordEventSubscribed()
	}

	kc.sendEvent(event, eventChan, msg)
}

// handleFetchError handles errors from fetching messages.
func (kc *Consumer) handleFetchError(err error, errorChan chan<- error) {
	kc.logger.Error("error fetching message from Kafka", logging.Field{Key: "error", Value: err})
	if errorChan != nil {
		select {
		case errorChan <- fmt.Errorf("fetch message failed: %w", err):
		case <-kc.ctx.Done():
		}
	}
	if kc.metrics != nil {
		kc.metrics.RecordError("subscribe", "fetch_error")
	}
}

// handleDeserializationError handles errors from deserializing messages.
func (kc *Consumer) handleDeserializationError(msg kafka.Message) {
	kc.logger.Warn("failed to deserialize message", logging.Field{Key: "error", Value: "deserialization error"})
	if kc.metrics != nil {
		kc.metrics.RecordError("subscribe", "deserialization_error")
	}
	kc.commitMessage(msg)
}

// commitMessage commits a message offset.
func (kc *Consumer) commitMessage(msg kafka.Message) {
	if err := kc.reader.CommitMessages(kc.ctx, msg); err != nil {
		kc.logger.Warn("failed to commit message", logging.Field{Key: "error", Value: err})
	}
}

// sendEvent sends an event to the event channel.
func (kc *Consumer) sendEvent(event *entity.Event, eventChan chan<- *entity.Event, msg kafka.Message) {
	select {
	case eventChan <- event:
		kc.commitMessage(msg)
	case <-kc.ctx.Done():
	}
}

// Unsubscribe gracefully stops consuming messages.
func (kc *Consumer) Unsubscribe(ctx context.Context) error {
	kc.mu.Lock()
	if !kc.isRunning {
		kc.mu.Unlock()
		return fmt.Errorf("consumer is not running")
	}

	// Cancel context to stop the consumer goroutine
	kc.cancel()
	kc.mu.Unlock()

	// Wait for consumer goroutine to finish with timeout
	done := make(chan struct{})
	go func() {
		kc.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Consumer stopped successfully
	case <-ctx.Done():
		kc.logger.Warn("timeout waiting for consumer to stop")
	}

	// Close the reader
	if err := kc.reader.Close(); err != nil {
		kc.logger.Error("failed to close Kafka reader", logging.Field{Key: "error", Value: err})
		return fmt.Errorf("failed to close Kafka reader: %w", err)
	}

	kc.logger.Info("consumer unsubscribed")
	return nil
}

// SetFilter sets a filter function for messages.
func (kc *Consumer) SetFilter(filter EventFilter) {
	kc.mu.Lock()
	defer kc.mu.Unlock()
	kc.filter = filter
}

// SetSerializer sets the event serializer for the consumer.
func (kc *Consumer) SetSerializer(serializer EventSerializer) {
	kc.serializer = serializer
}

// GetOffset returns the current offset for a partition.
func (kc *Consumer) GetOffset(partition int) int64 {
	return kc.offsetManager.GetOffset(partition)
}

// CommitOffset manually commits an offset for a partition.
func (kc *Consumer) CommitOffset(ctx context.Context, partition int, offset int64) error {
	kc.offsetManager.UpdateOffset(partition, offset)
	return nil
}

// OffsetManager implementation

// UpdateOffset updates the offset for a partition.
func (om *OffsetManager) UpdateOffset(partition int, offset int64) {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.offsets[partition] = offset
}

// GetOffset retrieves the offset for a partition.
func (om *OffsetManager) GetOffset(partition int) int64 {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return om.offsets[partition]
}

// GetAllOffsets returns a copy of all offsets.
func (om *OffsetManager) GetAllOffsets() map[int]int64 {
	om.mu.RLock()
	defer om.mu.RUnlock()

	result := make(map[int]int64)
	for k, v := range om.offsets {
		result[k] = v
	}
	return result
}

// Reset clears all offsets.
func (om *OffsetManager) Reset() {
	om.mu.Lock()
	defer om.mu.Unlock()
	om.offsets = make(map[int]int64)
}

// Close gracefully shuts down the consumer.
func (kc *Consumer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return kc.Unsubscribe(ctx)
}
