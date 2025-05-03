package repository

import (
	"context"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// EventRepository defines the contract for event persistence operations.
type EventRepository interface {
	// Store persists an event to the repository.
	// Returns an error if the event cannot be stored.
	Store(ctx context.Context, event *entity.Event) error

	// Get retrieves an event by its ID.
	// Returns the event or an error if not found or operation fails.
	Get(ctx context.Context, id string) (*entity.Event, error)

	// Query retrieves events based on filter and time range criteria.
	// Returns a slice of events matching the criteria or an error.
	Query(ctx context.Context, filter valueobject.EventFilter, timeRange *TimeRange) ([]*entity.Event, error)
}

// ConsumerRepository defines the contract for consumer persistence operations.
type ConsumerRepository interface {
	// Save persists a consumer to the repository.
	// If the consumer already exists, it should be updated.
	// Returns an error if the operation fails.
	Save(ctx context.Context, consumer *entity.Consumer) error

	// Get retrieves a consumer by its ID.
	// Returns the consumer or an error if not found or operation fails.
	Get(ctx context.Context, id string) (*entity.Consumer, error)

	// List retrieves all consumers matching the given group ID.
	// Returns a slice of consumers or an error.
	List(ctx context.Context, groupID string) ([]*entity.Consumer, error)

	// Delete removes a consumer from the repository.
	// Returns an error if the operation fails.
	Delete(ctx context.Context, id string) error
}

// SchemaRepository defines the contract for schema persistence operations.
type SchemaRepository interface {
	// Save persists a schema to the repository.
	// If the schema already exists, it should be updated.
	// Returns an error if the operation fails.
	Save(ctx context.Context, schema *entity.Schema) error

	// Get retrieves a schema by its ID.
	// Returns the schema or an error if not found or operation fails.
	Get(ctx context.Context, id string) (*entity.Schema, error)

	// GetByNameVersion retrieves a schema by name and version.
	// Returns the schema or an error if not found or operation fails.
	GetByNameVersion(ctx context.Context, name, version string) (*entity.Schema, error)

	// List retrieves all schemas with the given name, ordered by version.
	// Returns a slice of schemas or an error.
	List(ctx context.Context, name string) ([]*entity.Schema, error)

	// Delete removes a schema from the repository.
	// Returns an error if the operation fails.
	Delete(ctx context.Context, id string) error
}

// EventPublisher defines the contract for publishing events to a messaging system.
type EventPublisher interface {
	// Publish sends an event to the specified topic.
	// Returns an error if the event cannot be published.
	Publish(ctx context.Context, event *entity.Event, topic string) error

	// PublishBatch sends multiple events to the specified topic in a single operation.
	// Returns an error if any event in the batch cannot be published.
	PublishBatch(ctx context.Context, events []*entity.Event, topic string) error
}

// EventSubscriber defines the contract for subscribing to events from a messaging system.
type EventSubscriber interface {
	// Subscribe creates a subscription to events from the specified topic.
	// The filter determines which events are delivered to the consumer.
	// Returns a channel that will receive matching events and an error if subscription fails.
	Subscribe(
		ctx context.Context,
		topic string,
		group string,
		filter valueobject.EventFilter,
	) (<-chan *entity.Event, error)

	// Unsubscribe closes the subscription and stops receiving events.
	// Returns an error if the operation fails.
	Unsubscribe() error
}

// TimeRange represents a time interval for querying events.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewTimeRange creates a new TimeRange from the given start and end times.
func NewTimeRange(start, end time.Time) *TimeRange {
	return &TimeRange{
		Start: start,
		End:   end,
	}
}

// Duration returns the duration of the time range.
func (tr *TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// Contains returns true if the given time falls within this range.
func (tr *TimeRange) Contains(t time.Time) bool {
	return !t.Before(tr.Start) && !t.After(tr.End)
}
