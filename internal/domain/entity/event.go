package entity

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// Priority is an alias for valueobject.Priority for convenience.
type Priority = valueobject.Priority

// Priority constants re-exported from valueobject for backward compatibility.
const (
	PriorityUnspecified = valueobject.PriorityUnspecified
	PriorityLow         = valueobject.PriorityLow
	PriorityMedium      = valueobject.PriorityMedium
	PriorityHigh        = valueobject.PriorityHigh
	PriorityCritical    = valueobject.PriorityCritical
)

// Event is a domain entity that represents an event in the system.
// Events are immutable after creation to ensure data integrity and consistency.
type Event struct {
	id              string
	source          string
	eventType       string
	subject         string
	data            []byte
	dataContentType string
	schemaURL       string
	timestamp       time.Time
	metadata        map[string]string
	correlationID   string
	causationID     string
	partitionKey    string
	priority        Priority
	createdAt       time.Time
}

// NewEvent creates a new Event with the given parameters and validates it.
// Returns an error if the event fails validation.
func NewEvent(
	id string,
	source string,
	eventType string,
	subject string,
	data []byte,
	dataContentType string,
) (*Event, error) {
	e := &Event{
		id:              id,
		source:          source,
		eventType:       eventType,
		subject:         subject,
		data:            data,
		dataContentType: dataContentType,
		timestamp:       time.Now().UTC(),
		metadata:        make(map[string]string),
		priority:        PriorityMedium,
		createdAt:       time.Now().UTC(),
	}

	if err := e.Validate(); err != nil {
		return nil, err
	}

	return e, nil
}

// Validate checks if the event is valid according to business rules.
func (e *Event) Validate() error {
	if e.id == "" {
		return fmt.Errorf("event ID cannot be empty")
	}
	if e.source == "" {
		return fmt.Errorf("event source cannot be empty")
	}
	if e.eventType == "" {
		return fmt.Errorf("event type cannot be empty")
	}
	if len(e.data) == 0 {
		return fmt.Errorf("event data cannot be empty")
	}
	if e.dataContentType == "" {
		return fmt.Errorf("event data content type cannot be empty")
	}
	return nil
}

// ID returns the event ID.
func (e *Event) ID() string { return e.id }

// Source returns the event source.
func (e *Event) Source() string { return e.source }

// Type returns the event type.
func (e *Event) Type() string { return e.eventType }

// Subject returns the event subject.
func (e *Event) Subject() string { return e.subject }

// Data returns the event data directly (immutable, no copy needed).
func (e *Event) Data() []byte {
	return e.data
}

// DataCopy returns a copy of the event data for callers that need to modify it.
func (e *Event) DataCopy() []byte {
	dataCopy := make([]byte, len(e.data))
	copy(dataCopy, e.data)
	return dataCopy
}

// DataContentType returns the content type of the event data.
func (e *Event) DataContentType() string { return e.dataContentType }

// SchemaURL returns the schema URL if set.
func (e *Event) SchemaURL() string { return e.schemaURL }

// Timestamp returns the event timestamp.
func (e *Event) Timestamp() time.Time { return e.timestamp }

// Metadata returns the metadata map directly (immutable, no copy needed).
func (e *Event) Metadata() map[string]string {
	return e.metadata
}

// MetadataCopy returns a copy of the metadata map for callers that need to modify it.
func (e *Event) MetadataCopy() map[string]string {
	metadataCopy := make(map[string]string, len(e.metadata))
	for k, v := range e.metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

// CorrelationID returns the correlation ID.
func (e *Event) CorrelationID() string { return e.correlationID }

// CausationID returns the causation ID.
func (e *Event) CausationID() string { return e.causationID }

// PartitionKey returns the partition key.
func (e *Event) PartitionKey() string { return e.partitionKey }

// Priority returns the event priority.
func (e *Event) Priority() Priority { return e.priority }

// CreatedAt returns the event creation time.
func (e *Event) CreatedAt() time.Time { return e.createdAt }

// WithMetadata returns a new Event with the provided metadata merged.
func (e *Event) WithMetadata(metadata map[string]string) *Event {
	newEvent := e.copy()
	for k, v := range metadata {
		newEvent.metadata[k] = v
	}
	return newEvent
}

// WithCorrelation returns a new Event with the correlation and causation IDs set.
func (e *Event) WithCorrelation(correlationID, causationID string) *Event {
	newEvent := e.copy()
	newEvent.correlationID = correlationID
	newEvent.causationID = causationID
	return newEvent
}

// WithPriority returns a new Event with the specified priority.
func (e *Event) WithPriority(priority Priority) *Event {
	newEvent := e.copy()
	newEvent.priority = priority
	return newEvent
}

// WithPartitionKey returns a new Event with the specified partition key.
func (e *Event) WithPartitionKey(partitionKey string) *Event {
	newEvent := e.copy()
	newEvent.partitionKey = partitionKey
	return newEvent
}

// WithSchemaURL returns a new Event with the specified schema URL.
func (e *Event) WithSchemaURL(schemaURL string) *Event {
	newEvent := e.copy()
	newEvent.schemaURL = schemaURL
	return newEvent
}

// IsHighPriority returns true if the event priority is High or Critical.
func (e *Event) IsHighPriority() bool {
	return e.priority == PriorityHigh || e.priority == PriorityCritical
}

// Age returns the duration since the event was created.
func (e *Event) Age() time.Duration {
	return time.Since(e.createdAt)
}

// MarshalJSON implements the json.Marshaler interface for Event,
// enabling proper JSON serialization of the immutable entity.
func (e *Event) MarshalJSON() ([]byte, error) {
	type eventJSON struct {
		ID              string            `json:"id"`
		Source          string            `json:"source"`
		Type            string            `json:"type"`
		Subject         string            `json:"subject"`
		Data            json.RawMessage   `json:"data,omitempty"`
		DataContentType string            `json:"dataContentType"`
		SchemaURL       string            `json:"schemaUrl,omitempty"`
		Timestamp       time.Time         `json:"timestamp"`
		Metadata        map[string]string `json:"metadata,omitempty"`
		CorrelationID   string            `json:"correlationId,omitempty"`
		CausationID     string            `json:"causationId,omitempty"`
		PartitionKey    string            `json:"partitionKey,omitempty"`
		Priority        int               `json:"priority"`
	}
	return json.Marshal(eventJSON{
		ID:              e.id,
		Source:          e.source,
		Type:            e.eventType,
		Subject:         e.subject,
		Data:            e.data,
		DataContentType: e.dataContentType,
		SchemaURL:       e.schemaURL,
		Timestamp:       e.timestamp,
		Metadata:        e.metadata,
		CorrelationID:   e.correlationID,
		CausationID:     e.causationID,
		PartitionKey:    e.partitionKey,
		Priority:        int(e.priority),
	})
}

// copy creates a shallow copy of the event, sharing immutable data but creating new metadata map.
// Data is immutable and never modified after creation, so we share the reference for performance.
// Metadata must be a new map because WithMetadata() modifies it after calling copy().
func (e *Event) copy() *Event {
	newEvent := &Event{
		id:              e.id,
		source:          e.source,
		eventType:       e.eventType,
		subject:         e.subject,
		data:            e.data, // Share reference, immutable and never modified after creation
		dataContentType: e.dataContentType,
		schemaURL:       e.schemaURL,
		timestamp:       e.timestamp,
		correlationID:   e.correlationID,
		causationID:     e.causationID,
		partitionKey:    e.partitionKey,
		priority:        e.priority,
		createdAt:       e.createdAt,
		metadata:        make(map[string]string, len(e.metadata)),
	}
	// Copy metadata entries into new map
	for k, v := range e.metadata {
		newEvent.metadata[k] = v
	}
	return newEvent
}
