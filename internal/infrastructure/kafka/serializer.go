package kafka

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// EventSerializer defines the interface for event serialization/deserialization.
type EventSerializer interface {
	Serialize(event *entity.Event) ([]byte, error)
	Deserialize(data []byte) (*entity.Event, error)
}

// eventDeserializationDTO is a data transfer object for deserializing events from JSON.
// It matches the structure produced by Event.MarshalJSON().
type eventDeserializationDTO struct {
	ID              string            `json:"id"`
	Source          string            `json:"source"`
	Type            string            `json:"type"`
	Subject         string            `json:"subject"`
	Data            json.RawMessage   `json:"data,omitempty"`
	DataContentType string            `json:"dataContentType"`
	SchemaURL       string            `json:"schemaUrl,omitempty"`
	Timestamp       string            `json:"timestamp"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	CorrelationID   string            `json:"correlationId,omitempty"`
	CausationID     string            `json:"causationId,omitempty"`
	PartitionKey    string            `json:"partitionKey,omitempty"`
	Priority        int               `json:"priority"`
}

// JSONSerializer implements EventSerializer using JSON encoding.
// Uses a sync.Pool to reuse bytes.Buffer instances for efficient serialization.
type JSONSerializer struct {
	bufferPool *sync.Pool
}

// NewJSONSerializer creates a new JSONSerializer with a buffer pool.
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{
		bufferPool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
}

// Serialize encodes an event to JSON bytes using a pooled buffer.
func (s *JSONSerializer) Serialize(event *entity.Event) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("event cannot be nil")
	}

	// Get a buffer from the pool
	bufObj := s.bufferPool.Get()
	if bufObj == nil {
		return nil, fmt.Errorf("failed to get buffer from pool")
	}
	buf, ok := bufObj.(*bytes.Buffer)
	if !ok {
		return nil, fmt.Errorf("invalid buffer type from pool")
	}
	defer func() {
		buf.Reset()
		s.bufferPool.Put(buf)
	}()

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(event); err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	// Return a copy of the data since the buffer will be reused
	result := make([]byte, buf.Len()-1) // -1 to remove trailing newline from Encode
	copy(result, buf.Bytes()[:buf.Len()-1])
	return result, nil
}

// Deserialize decodes JSON bytes to an event.
// It unmarshals into a DTO first, then reconstructs the Event entity using NewEvent()
// and applies additional fields via builder methods.
func (s *JSONSerializer) Deserialize(data []byte) (*entity.Event, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	var dto eventDeserializationDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Create the event using the constructor with required fields
	event, err := entity.NewEvent(
		dto.ID,
		dto.Source,
		dto.Type,
		dto.Subject,
		[]byte(dto.Data),
		dto.DataContentType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create event entity: %w", err)
	}

	// Apply optional fields using builder methods
	if dto.SchemaURL != "" {
		event = event.WithSchemaURL(dto.SchemaURL)
	}

	if dto.CorrelationID != "" || dto.CausationID != "" {
		event = event.WithCorrelation(dto.CorrelationID, dto.CausationID)
	}

	if len(dto.Metadata) > 0 {
		event = event.WithMetadata(dto.Metadata)
	}

	if dto.Priority > 0 {
		event = event.WithPriority(entity.Priority(dto.Priority))
	}

	if dto.PartitionKey != "" {
		event = event.WithPartitionKey(dto.PartitionKey)
	}

	return event, nil
}

// ProtobufSerializer implements EventSerializer using Protobuf encoding.
// This is a placeholder that will use generated protobuf code.
type ProtobufSerializer struct{}

// NewProtobufSerializer creates a new ProtobufSerializer.
func NewProtobufSerializer() *ProtobufSerializer {
	return &ProtobufSerializer{}
}

// Serialize encodes an event to Protobuf bytes.
func (s *ProtobufSerializer) Serialize(event *entity.Event) ([]byte, error) {
	if event == nil {
		return nil, fmt.Errorf("event cannot be nil")
	}

	return nil, fmt.Errorf("protobuf serialization not implemented")
}

// Deserialize decodes Protobuf bytes to an event.
func (s *ProtobufSerializer) Deserialize(data []byte) (*entity.Event, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	return nil, fmt.Errorf("protobuf deserialization not implemented")
}

// SerializerFactory creates appropriate serializer based on content type.
func SerializerFactory(contentType string) (EventSerializer, error) {
	switch contentType {
	case "application/json":
		return NewJSONSerializer(), nil
	case "application/protobuf", "application/octet-stream":
		return NewProtobufSerializer(), nil
	default:
		// Default to JSON
		return NewJSONSerializer(), nil
	}
}
