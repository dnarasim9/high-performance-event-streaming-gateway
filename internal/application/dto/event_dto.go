package dto

import (
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// IngestEventRequest is a DTO for ingesting a single event.
type IngestEventRequest struct {
	Source          string            `json:"source"`
	Type            string            `json:"type"`
	Subject         string            `json:"subject"`
	Data            []byte            `json:"data"`
	DataContentType string            `json:"dataContentType"`
	SchemaURL       string            `json:"schemaURL,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CorrelationID   string            `json:"correlationID,omitempty"`
	CausationID     string            `json:"causationID,omitempty"`
	PartitionKey    string            `json:"partitionKey,omitempty"`
	Priority        int               `json:"priority,omitempty"`
}

// ToEntity converts an IngestEventRequest to a domain Event entity.
// Returns an error if the event fails validation.
func (r *IngestEventRequest) ToEntity(id string) (*entity.Event, error) {
	event, err := entity.NewEvent(
		id,
		r.Source,
		r.Type,
		r.Subject,
		r.Data,
		r.DataContentType,
	)
	if err != nil {
		return nil, err
	}

	// Apply optional fields
	if r.SchemaURL != "" {
		event = event.WithSchemaURL(r.SchemaURL)
	}

	if len(r.Metadata) > 0 {
		event = event.WithMetadata(r.Metadata)
	}

	if r.CorrelationID != "" || r.CausationID != "" {
		event = event.WithCorrelation(r.CorrelationID, r.CausationID)
	}

	if r.PartitionKey != "" {
		event = event.WithPartitionKey(r.PartitionKey)
	}

	if r.Priority > 0 && r.Priority <= int(entity.PriorityCritical) {
		event = event.WithPriority(entity.Priority(r.Priority))
	}

	return event, nil
}

// IngestEventResponse is a DTO for the response to an event ingestion request.
type IngestEventResponse struct {
	EventID   string    `json:"eventID"`
	Timestamp time.Time `json:"timestamp"`
}

// FromEntity creates an IngestEventResponse from a domain Event entity.
func FromEntity(event *entity.Event) *IngestEventResponse {
	return &IngestEventResponse{
		EventID:   event.ID(),
		Timestamp: event.Timestamp(),
	}
}

// SubscribeRequest is a DTO for subscribing to events.
type SubscribeRequest struct {
	Filter     *EventFilterDTO `json:"filter,omitempty"`
	GroupID    string          `json:"groupID"`
	ConsumerID string          `json:"consumerID"`
}

// EventFilterDTO represents filter criteria for subscribing to events.
type EventFilterDTO struct {
	SourcePattern   string            `json:"sourcePattern,omitempty"`
	TypePattern     string            `json:"typePattern,omitempty"`
	SubjectPattern  string            `json:"subjectPattern,omitempty"`
	MetadataFilters map[string]string `json:"metadataFilters,omitempty"`
	MinPriority     int               `json:"minPriority,omitempty"`
}

// ToValueObject converts an EventFilterDTO to a domain EventFilter value object.
// Returns an error if the filter contains invalid patterns.
func (f *EventFilterDTO) ToValueObject() (valueobject.EventFilter, error) {
	return valueobject.NewEventFilter(
		f.SourcePattern,
		f.TypePattern,
		f.SubjectPattern,
		f.MetadataFilters,
		entity.Priority(f.MinPriority),
	)
}

// ReplayRequest is a DTO for replaying events from a time range.
type ReplayRequest struct {
	StartTime time.Time       `json:"startTime"`
	EndTime   time.Time       `json:"endTime"`
	Filter    *EventFilterDTO `json:"filter,omitempty"`
	Limit     int             `json:"limit,omitempty"`
}

// FromEntityEvent creates an IngestEventResponse from a domain Event entity.
// This is an alias for the FromEntity function for consistency.
func FromEntityEvent(event *entity.Event) *IngestEventResponse {
	return FromEntity(event)
}
