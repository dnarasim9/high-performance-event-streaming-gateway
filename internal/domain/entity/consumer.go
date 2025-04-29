package entity

import (
	"fmt"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// ConsumerStatus represents the status of a consumer.
type ConsumerStatus int

// ConsumerStatus constants define the supported consumer statuses.
const (
	ConsumerStatusActive ConsumerStatus = iota
	ConsumerStatusInactive
	ConsumerStatusDraining
)

// String returns the string representation of ConsumerStatus.
func (s ConsumerStatus) String() string {
	switch s {
	case ConsumerStatusActive:
		return "Active"
	case ConsumerStatusInactive:
		return "Inactive"
	case ConsumerStatusDraining:
		return "Draining"
	default:
		return "Unknown"
	}
}

// Consumer is a domain entity that represents a consumer in the system.
// Consumers subscribe to events based on filters and apply transformations.
type Consumer struct {
	id             string
	name           string
	groupID        string
	filter         valueobject.EventFilter
	transformation valueobject.Transformation
	status         ConsumerStatus
	createdAt      time.Time
	updatedAt      time.Time
}

// NewConsumer creates a new Consumer with the given parameters.
// Returns an error if any required field is invalid.
func NewConsumer(
	id string,
	name string,
	groupID string,
	filter valueobject.EventFilter,
	transformation valueobject.Transformation,
) (*Consumer, error) {
	if id == "" {
		return nil, fmt.Errorf("consumer ID cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("consumer name cannot be empty")
	}
	if groupID == "" {
		return nil, fmt.Errorf("consumer group ID cannot be empty")
	}

	if err := transformation.Validate(); err != nil {
		return nil, fmt.Errorf("invalid transformation: %w", err)
	}

	now := time.Now().UTC()
	return &Consumer{
		id:             id,
		name:           name,
		groupID:        groupID,
		filter:         filter,
		transformation: transformation,
		status:         ConsumerStatusInactive,
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

// ID returns the consumer ID.
func (c *Consumer) ID() string {
	return c.id
}

// Name returns the consumer name.
func (c *Consumer) Name() string {
	return c.name
}

// GroupID returns the consumer group ID.
func (c *Consumer) GroupID() string {
	return c.groupID
}

// Filter returns the event filter.
func (c *Consumer) Filter() valueobject.EventFilter {
	return c.filter
}

// Transformation returns the transformation configuration.
func (c *Consumer) Transformation() valueobject.Transformation {
	return c.transformation
}

// Status returns the current consumer status.
func (c *Consumer) Status() ConsumerStatus {
	return c.status
}

// CreatedAt returns the creation timestamp.
func (c *Consumer) CreatedAt() time.Time {
	return c.createdAt
}

// UpdatedAt returns the last update timestamp.
func (c *Consumer) UpdatedAt() time.Time {
	return c.updatedAt
}

// Activate activates the consumer, changing its status to Active.
func (c *Consumer) Activate() error {
	if c.status == ConsumerStatusDraining {
		return fmt.Errorf("cannot activate a draining consumer")
	}
	c.status = ConsumerStatusActive
	c.updatedAt = time.Now().UTC()
	return nil
}

// Deactivate deactivates the consumer, changing its status to Inactive.
func (c *Consumer) Deactivate() error {
	if c.status == ConsumerStatusDraining {
		return fmt.Errorf("cannot deactivate a draining consumer")
	}
	c.status = ConsumerStatusInactive
	c.updatedAt = time.Now().UTC()
	return nil
}

// Drain transitions the consumer to a draining state, allowing it to
// finish processing existing messages before becoming fully inactive.
func (c *Consumer) Drain() {
	c.status = ConsumerStatusDraining
	c.updatedAt = time.Now().UTC()
}

// UpdateFilter updates the event filter for this consumer.
func (c *Consumer) UpdateFilter(filter valueobject.EventFilter) error {
	if c.status == ConsumerStatusDraining {
		return fmt.Errorf("cannot update filter on a draining consumer")
	}
	c.filter = filter
	c.updatedAt = time.Now().UTC()
	return nil
}

// MatchesEvent returns true if the given event matches this consumer's filter.
func (c *Consumer) MatchesEvent(event *Event) bool {
	return c.filter.Matches(event)
}

// IsActive returns true if the consumer is in Active status.
func (c *Consumer) IsActive() bool {
	return c.status == ConsumerStatusActive
}

// IsInactive returns true if the consumer is in Inactive status.
func (c *Consumer) IsInactive() bool {
	return c.status == ConsumerStatusInactive
}

// IsDraining returns true if the consumer is in Draining status.
func (c *Consumer) IsDraining() bool {
	return c.status == ConsumerStatusDraining
}
