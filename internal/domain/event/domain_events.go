package event

import (
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// DomainEvent is the interface that all domain events must implement.
// Domain events represent significant business occurrences that have already happened.
type DomainEvent interface {
	// EventType returns the type of the domain event.
	EventType() string

	// OccurredAt returns the time when the event occurred.
	OccurredAt() time.Time

	// AggregateID returns the ID of the aggregate that this event is associated with.
	AggregateID() string
}

// Ingested is a domain event that indicates an event has been ingested into the system.
type Ingested struct {
	eventID     string
	source      string
	eventType   string
	occurredAt  time.Time
	aggregateID string
}

// NewIngested creates a new Ingested domain event.
func NewIngested(event *entity.Event) *Ingested {
	return &Ingested{
		eventID:     event.ID(),
		source:      event.Source(),
		eventType:   event.Type(),
		occurredAt:  time.Now().UTC(),
		aggregateID: event.ID(),
	}
}

// EventType returns the type of the domain event.
func (e *Ingested) EventType() string {
	return "Ingested"
}

// OccurredAt returns the time when the event occurred.
func (e *Ingested) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the ID of the aggregate that this event is associated with.
func (e *Ingested) AggregateID() string {
	return e.aggregateID
}

// EventID returns the ID of the ingested event.
func (e *Ingested) EventID() string {
	return e.eventID
}

// Source returns the source of the ingested event.
func (e *Ingested) Source() string {
	return e.source
}

// Type returns the type of the ingested event.
func (e *Ingested) Type() string {
	return e.eventType
}

// Published is a domain event that indicates an event has been published to consumers.
type Published struct {
	eventID     string
	topic       string
	occurredAt  time.Time
	aggregateID string
}

// NewPublished creates a new Published domain event.
func NewPublished(event *entity.Event, topic string) *Published {
	return &Published{
		eventID:     event.ID(),
		topic:       topic,
		occurredAt:  time.Now().UTC(),
		aggregateID: event.ID(),
	}
}

// EventType returns the type of the domain event.
func (e *Published) EventType() string {
	return "Published"
}

// OccurredAt returns the time when the event occurred.
func (e *Published) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the ID of the aggregate that this event is associated with.
func (e *Published) AggregateID() string {
	return e.aggregateID
}

// EventID returns the ID of the published event.
func (e *Published) EventID() string {
	return e.eventID
}

// Topic returns the topic to which the event was published.
func (e *Published) Topic() string {
	return e.topic
}

// ConsumerRegistered is a domain event that indicates a new consumer has been registered.
type ConsumerRegistered struct {
	consumerID  string
	name        string
	groupID     string
	occurredAt  time.Time
	aggregateID string
}

// NewConsumerRegistered creates a new ConsumerRegistered domain event.
func NewConsumerRegistered(consumer *entity.Consumer) *ConsumerRegistered {
	return &ConsumerRegistered{
		consumerID:  consumer.ID(),
		name:        consumer.Name(),
		groupID:     consumer.GroupID(),
		occurredAt:  time.Now().UTC(),
		aggregateID: consumer.ID(),
	}
}

// EventType returns the type of the domain event.
func (e *ConsumerRegistered) EventType() string {
	return "ConsumerRegistered"
}

// OccurredAt returns the time when the event occurred.
func (e *ConsumerRegistered) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the ID of the aggregate that this event is associated with.
func (e *ConsumerRegistered) AggregateID() string {
	return e.aggregateID
}

// ConsumerID returns the ID of the registered consumer.
func (e *ConsumerRegistered) ConsumerID() string {
	return e.consumerID
}

// Name returns the name of the registered consumer.
func (e *ConsumerRegistered) Name() string {
	return e.name
}

// GroupID returns the group ID of the registered consumer.
func (e *ConsumerRegistered) GroupID() string {
	return e.groupID
}

// ConsumerDeregistered is a domain event that indicates a consumer has been deregistered.
type ConsumerDeregistered struct {
	consumerID  string
	groupID     string
	occurredAt  time.Time
	aggregateID string
}

// NewConsumerDeregistered creates a new ConsumerDeregistered domain event.
func NewConsumerDeregistered(consumer *entity.Consumer) *ConsumerDeregistered {
	return &ConsumerDeregistered{
		consumerID:  consumer.ID(),
		groupID:     consumer.GroupID(),
		occurredAt:  time.Now().UTC(),
		aggregateID: consumer.ID(),
	}
}

// EventType returns the type of the domain event.
func (e *ConsumerDeregistered) EventType() string {
	return "ConsumerDeregistered"
}

// OccurredAt returns the time when the event occurred.
func (e *ConsumerDeregistered) OccurredAt() time.Time {
	return e.occurredAt
}

// AggregateID returns the ID of the aggregate that this event is associated with.
func (e *ConsumerDeregistered) AggregateID() string {
	return e.aggregateID
}

// ConsumerID returns the ID of the deregistered consumer.
func (e *ConsumerDeregistered) ConsumerID() string {
	return e.consumerID
}

// GroupID returns the group ID of the deregistered consumer.
func (e *ConsumerDeregistered) GroupID() string {
	return e.groupID
}
