package port

import (
	"context"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// EventIngestionUseCase defines the contract for event ingestion operations.
type EventIngestionUseCase interface {
	// IngestEvent processes a single event for ingestion.
	// Returns the ID of the ingested event or an error.
	IngestEvent(ctx context.Context, request *dto.IngestEventRequest) (string, error)

	// IngestEventBatch processes multiple events for ingestion.
	// Returns a slice of ingested event IDs or an error.
	IngestEventBatch(ctx context.Context, requests []*dto.IngestEventRequest) ([]string, error)
}

// EventSubscriptionUseCase defines the contract for event subscription operations.
type EventSubscriptionUseCase interface {
	// Subscribe creates a new subscription to events matching the specified filter.
	// Returns a channel that receives matching events or an error.
	Subscribe(ctx context.Context, request *dto.SubscribeRequest) (<-chan *entity.Event, error)

	// Unsubscribe cancels an active subscription.
	// Returns an error if the subscription cannot be canceled.
	Unsubscribe(ctx context.Context, subscriptionID string) error
}

// EventReplayUseCase defines the contract for event replay operations.
type EventReplayUseCase interface {
	// ReplayEvents retrieves and streams events from a specified time range.
	// Optionally applies filters and limits the number of returned events.
	// Returns a channel that receives replayed events or an error.
	ReplayEvents(ctx context.Context, request *dto.ReplayRequest) (<-chan *entity.Event, error)
}

// ConsumerManagementUseCase defines the contract for consumer management operations.
type ConsumerManagementUseCase interface {
	// Register registers a new consumer in the system.
	// Returns the ID of the registered consumer or an error.
	Register(ctx context.Context, request *dto.RegisterConsumerRequest) (string, error)

	// Deregister removes a consumer from the system.
	// Returns an error if the consumer cannot be deregistered.
	Deregister(ctx context.Context, consumerID string) error

	// Get retrieves a consumer by its ID.
	// Returns the consumer details or an error.
	Get(ctx context.Context, consumerID string) (*dto.ConsumerResponse, error)

	// List retrieves all consumers in a consumer group.
	// Returns a slice of consumer details or an error.
	List(ctx context.Context, groupID string) ([]*dto.ConsumerResponse, error)
}

// SchemaManagementUseCase defines the contract for schema management operations.
type SchemaManagementUseCase interface {
	// Register registers a new schema in the system.
	// Returns the ID of the registered schema or an error.
	Register(ctx context.Context, request *dto.RegisterSchemaRequest) (string, error)

	// Get retrieves a schema by its ID.
	// Returns the schema details or an error.
	Get(ctx context.Context, schemaID string) (*dto.SchemaResponse, error)

	// Validate validates event data against a specific schema.
	// Returns an error if the data does not conform to the schema.
	Validate(ctx context.Context, request *dto.ValidateRequest) (*dto.ValidateResponse, error)

	// List retrieves all versions of a schema by name.
	// Returns a slice of schema details or an error.
	List(ctx context.Context, schemaName string) ([]*dto.SchemaResponse, error)
}
