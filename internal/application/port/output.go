package port

import (
	"context"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/repository"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// EventStore defines the contract for event storage operations.
// Extends the repository.EventRepository interface.
type EventStore interface {
	repository.EventRepository
}

// ConsumerRepository defines the contract for consumer persistence operations.
type ConsumerRepository interface {
	Save(ctx context.Context, consumer *entity.Consumer) error
	Get(ctx context.Context, id string) (*entity.Consumer, error)
	List(ctx context.Context, groupID string) ([]*entity.Consumer, error)
	Delete(ctx context.Context, id string) error
}

// SchemaRepository defines the contract for schema persistence operations.
type SchemaRepository interface {
	Save(ctx context.Context, schema *entity.Schema) error
	Get(ctx context.Context, id string) (*entity.Schema, error)
	GetByNameVersion(ctx context.Context, name, version string) (*entity.Schema, error)
	List(ctx context.Context, name string) ([]*entity.Schema, error)
	Delete(ctx context.Context, id string) error
}

// EventPublisher defines the contract for publishing events to a message broker.
type EventPublisher interface {
	// Publish sends a single event to the specified topic.
	// Returns an error if the event cannot be published.
	Publish(ctx context.Context, event *entity.Event, topic string) error

	// PublishBatch sends multiple events to the specified topic in a batch.
	// Returns an error if any event in the batch cannot be published.
	PublishBatch(ctx context.Context, events []*entity.Event, topic string) error
}

// EventSubscriber defines the contract for subscribing to events from a message broker.
type EventSubscriber interface {
	// Subscribe creates a subscription to events from the specified topic.
	// Returns a channel that receives matching events and an error if subscription fails.
	Subscribe(
		ctx context.Context,
		topic string,
		group string,
		filter valueobject.EventFilter,
	) (<-chan *entity.Event, error)

	// Unsubscribe closes an active subscription.
	// Returns an error if the subscription cannot be closed.
	Unsubscribe() error

	// Close shuts down the subscriber and releases all resources.
	// Returns an error if the shutdown fails.
	Close() error
}

// MessageBroker defines the contract for publishing and subscribing to events via a message broker.
// It composes the EventPublisher and EventSubscriber interfaces for backward compatibility.
type MessageBroker interface {
	EventPublisher
	EventSubscriber
}

// SchemaValidator defines the contract for validating event data against schemas.
type SchemaValidator interface {
	// Validate checks if the provided data conforms to the specified schema.
	// Returns an error if validation fails.
	Validate(ctx context.Context, schemaID string, data []byte) error
}

// TransformationEngine defines the contract for applying transformations to events.
type TransformationEngine interface {
	// Transform applies a transformation to an event.
	// Returns a new transformed event or an error if transformation fails.
	Transform(ctx context.Context, event *entity.Event, transformation valueobject.Transformation) (*entity.Event, error)
}

// MetricsCollector defines the contract for recording operational metrics.
type MetricsCollector interface {
	// RecordIngestion records metrics for event ingestion.
	// eventCount: number of events ingested
	// duration: time taken for ingestion in milliseconds
	RecordIngestion(eventCount int, durationMS float64)

	// RecordPublish records metrics for event publishing.
	// eventCount: number of events published
	// duration: time taken for publishing in milliseconds
	RecordPublish(eventCount int, durationMS float64)

	// RecordSubscription records metrics for event subscription.
	// consumerID: identifier of the subscribing consumer
	// groupID: identifier of the consumer group
	RecordSubscription(consumerID, groupID string)

	// RecordLatency records the latency of an operation.
	// operation: name of the operation
	// durationMS: latency in milliseconds
	RecordLatency(operation string, durationMS float64)

	// RecordError records that an error occurred.
	// operation: name of the operation where the error occurred
	// errorType: classification of the error
	RecordError(operation, errorType string)
}
