package dto

import (
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

// RegisterConsumerRequest is a DTO for registering a new consumer.
type RegisterConsumerRequest struct {
	Name            string                   `json:"name"`
	GroupID         string                   `json:"groupID"`
	FilterConfig    *EventFilterDTO          `json:"filterConfig,omitempty"`
	TransformConfig *TransformationConfigDTO `json:"transformConfig,omitempty"`
}

// TransformationConfigDTO represents a transformation configuration.
type TransformationConfigDTO struct {
	Type       string `json:"type"`
	Expression string `json:"expression,omitempty"`
}

// ToValueObject converts a TransformationConfigDTO to a domain Transformation value object.
// Returns an error if the transformation is invalid.
func (t *TransformationConfigDTO) ToValueObject() (valueobject.Transformation, error) {
	var transformationType valueobject.TransformationType
	switch t.Type {
	case "JQ":
		transformationType = valueobject.TransformationTypeJQ
	case "CEL":
		transformationType = valueobject.TransformationTypeCEL
	case "Template":
		transformationType = valueobject.TransformationTypeTemplate
	default:
		transformationType = valueobject.TransformationTypeNone
	}

	return valueobject.NewTransformation(transformationType, t.Expression)
}

// ConsumerResponse is a DTO for consumer details in responses.
type ConsumerResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	GroupID   string    `json:"groupID"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromEntityConsumer creates a ConsumerResponse from a domain Consumer entity.
func FromEntityConsumer(consumer *entity.Consumer) *ConsumerResponse {
	return &ConsumerResponse{
		ID:        consumer.ID(),
		Name:      consumer.Name(),
		GroupID:   consumer.GroupID(),
		Status:    consumer.Status().String(),
		CreatedAt: consumer.CreatedAt(),
		UpdatedAt: consumer.UpdatedAt(),
	}
}

// FromEntityConsumers creates multiple ConsumerResponse DTOs from domain Consumer entities.
func FromEntityConsumers(consumers []*entity.Consumer) []*ConsumerResponse {
	responses := make([]*ConsumerResponse, len(consumers))
	for i, consumer := range consumers {
		responses[i] = FromEntityConsumer(consumer)
	}
	return responses
}
