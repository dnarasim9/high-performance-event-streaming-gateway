package kafka

import (
	"testing"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// TestJSONSerializer_Serialize_Success tests successful JSON serialization
func TestJSONSerializer_Serialize_Success(t *testing.T) {
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)

	if err != nil {
		t.Errorf("Serialize() error = %v, want nil", err)
	}
	if len(data) == 0 {
		t.Error("Serialize() returned empty data")
	}
}

// TestJSONSerializer_Serialize_NilEvent tests serialization of nil event
func TestJSONSerializer_Serialize_NilEvent(t *testing.T) {
	serializer := NewJSONSerializer()
	_, err := serializer.Serialize(nil)

	if err == nil {
		t.Error("Serialize() with nil event expected error, got nil")
	}
}

// TestJSONSerializer_Serialize_WithMetadata tests serialization with metadata
func TestJSONSerializer_Serialize_WithMetadata(t *testing.T) {
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	event = event.WithMetadata(map[string]string{
		"env":    "test",
		"region": "us-west",
	})

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)

	if err != nil {
		t.Errorf("Serialize() error = %v, want nil", err)
	}
	if len(data) == 0 {
		t.Error("Serialize() returned empty data")
	}
}

// TestJSONSerializer_Serialize_WithCorrelation tests serialization with correlation IDs
func TestJSONSerializer_Serialize_WithCorrelation(t *testing.T) {
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	event = event.WithCorrelation("corr-123", "cause-456")

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)

	if err != nil {
		t.Errorf("Serialize() error = %v, want nil", err)
	}
	if len(data) == 0 {
		t.Error("Serialize() returned empty data")
	}
}

// TestJSONSerializer_Serialize_WithSchemaURL tests serialization with schema URL
func TestJSONSerializer_Serialize_WithSchemaURL(t *testing.T) {
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	event = event.WithSchemaURL("https://example.com/schema.json")

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)

	if err != nil {
		t.Errorf("Serialize() error = %v, want nil", err)
	}
	if len(data) == 0 {
		t.Error("Serialize() returned empty data")
	}
}

// TestJSONSerializer_Serialize_WithPriority tests serialization with priority
func TestJSONSerializer_Serialize_WithPriority(t *testing.T) {
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	event = event.WithPriority(entity.PriorityHigh)

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)

	if err != nil {
		t.Errorf("Serialize() error = %v, want nil", err)
	}
	if len(data) == 0 {
		t.Error("Serialize() returned empty data")
	}
}

// TestJSONSerializer_Deserialize_Success tests successful JSON deserialization
func TestJSONSerializer_Deserialize_Success(t *testing.T) {
	// First serialize an event
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Now deserialize it
	deserializedEvent, err := serializer.Deserialize(data)
	if err != nil {
		t.Errorf("Deserialize() error = %v, want nil", err)
	}
	if deserializedEvent == nil {
		t.Error("Deserialize() returned nil")
	}
	if deserializedEvent.ID() != event.ID() {
		t.Errorf("Deserialize() ID = %s, want %s", deserializedEvent.ID(), event.ID())
	}
	if deserializedEvent.Source() != event.Source() {
		t.Errorf("Deserialize() Source = %s, want %s", deserializedEvent.Source(), event.Source())
	}
	if deserializedEvent.Type() != event.Type() {
		t.Errorf("Deserialize() Type = %s, want %s", deserializedEvent.Type(), event.Type())
	}
}

// TestJSONSerializer_Deserialize_EmptyData tests deserialization of empty data
func TestJSONSerializer_Deserialize_EmptyData(t *testing.T) {
	serializer := NewJSONSerializer()
	_, err := serializer.Deserialize([]byte{})

	if err == nil {
		t.Error("Deserialize() with empty data expected error, got nil")
	}
}

// TestJSONSerializer_Deserialize_InvalidJSON tests deserialization of invalid JSON
func TestJSONSerializer_Deserialize_InvalidJSON(t *testing.T) {
	serializer := NewJSONSerializer()
	_, err := serializer.Deserialize([]byte(`not valid json`))

	if err == nil {
		t.Error("Deserialize() with invalid JSON expected error, got nil")
	}
}

// TestJSONSerializer_Deserialize_MissingRequiredFields tests deserialization with missing required fields
func TestJSONSerializer_Deserialize_MissingRequiredFields(t *testing.T) {
	serializer := NewJSONSerializer()
	invalidJSON := []byte(`{"id": "event-1"}`)

	_, err := serializer.Deserialize(invalidJSON)
	if err == nil {
		t.Error("Deserialize() with missing required fields expected error, got nil")
	}
}

// TestJSONSerializer_Deserialize_WithMetadata tests deserialization with metadata
func TestJSONSerializer_Deserialize_WithMetadata(t *testing.T) {
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	event = event.WithMetadata(map[string]string{
		"env":    "test",
		"region": "us-west",
	})

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	deserializedEvent, err := serializer.Deserialize(data)
	if err != nil {
		t.Errorf("Deserialize() error = %v, want nil", err)
	}

	if deserializedEvent.Metadata() == nil {
		t.Error("Deserialize() metadata is nil")
	}
	if len(deserializedEvent.Metadata()) != 2 {
		t.Errorf("Deserialize() metadata length = %d, want 2", len(deserializedEvent.Metadata()))
	}
}

// TestJSONSerializer_Deserialize_WithCorrelation tests deserialization with correlation
func TestJSONSerializer_Deserialize_WithCorrelation(t *testing.T) {
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	event = event.WithCorrelation("corr-123", "cause-456")

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	deserializedEvent, err := serializer.Deserialize(data)
	if err != nil {
		t.Errorf("Deserialize() error = %v, want nil", err)
	}

	if deserializedEvent.CorrelationID() != "corr-123" {
		t.Errorf("Deserialize() CorrelationID = %s, want corr-123", deserializedEvent.CorrelationID())
	}
	if deserializedEvent.CausationID() != "cause-456" {
		t.Errorf("Deserialize() CausationID = %s, want cause-456", deserializedEvent.CausationID())
	}
}

// TestJSONSerializer_Deserialize_WithSchemaURL tests deserialization with schema URL
func TestJSONSerializer_Deserialize_WithSchemaURL(t *testing.T) {
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	event = event.WithSchemaURL("https://example.com/schema.json")

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	deserializedEvent, err := serializer.Deserialize(data)
	if err != nil {
		t.Errorf("Deserialize() error = %v, want nil", err)
	}

	if deserializedEvent.SchemaURL() != "https://example.com/schema.json" {
		t.Errorf("Deserialize() SchemaURL = %s, want https://example.com/schema.json", deserializedEvent.SchemaURL())
	}
}

// TestJSONSerializer_Deserialize_WithPriority tests deserialization with priority
func TestJSONSerializer_Deserialize_WithPriority(t *testing.T) {
	event, err := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	event = event.WithPriority(entity.PriorityHigh)

	serializer := NewJSONSerializer()
	data, err := serializer.Serialize(event)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	deserializedEvent, err := serializer.Deserialize(data)
	if err != nil {
		t.Errorf("Deserialize() error = %v, want nil", err)
	}

	if deserializedEvent.Priority() != entity.PriorityHigh {
		t.Errorf("Deserialize() Priority = %v, want PriorityHigh", deserializedEvent.Priority())
	}
}

// TestJSONSerializer_RoundTrip tests complete round-trip serialization/deserialization
func TestJSONSerializer_RoundTrip(t *testing.T) {
	original, err := entity.NewEvent(
		"event-123",
		"api-source",
		"user.created",
		"/users/123",
		[]byte(`{"name": "John", "email": "john@example.com"}`),
		"application/json",
	)
	if err != nil {
		t.Fatalf("Failed to create test event: %v", err)
	}

	original = original.
		WithMetadata(map[string]string{"env": "production"}).
		WithCorrelation("corr-789", "cause-101").
		WithSchemaURL("https://example.com/schema.json").
		WithPriority(entity.PriorityHigh).
		WithPartitionKey("user-123")

	serializer := NewJSONSerializer()

	// Serialize
	data, err := serializer.Serialize(original)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Deserialize
	restored, err := serializer.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}

	// Verify all fields
	if restored.ID() != original.ID() {
		t.Errorf("Round-trip failed: ID mismatch")
	}
	if restored.Source() != original.Source() {
		t.Errorf("Round-trip failed: Source mismatch")
	}
	if restored.Type() != original.Type() {
		t.Errorf("Round-trip failed: Type mismatch")
	}
	if restored.Subject() != original.Subject() {
		t.Errorf("Round-trip failed: Subject mismatch")
	}
	if restored.CorrelationID() != original.CorrelationID() {
		t.Errorf("Round-trip failed: CorrelationID mismatch")
	}
	if restored.CausationID() != original.CausationID() {
		t.Errorf("Round-trip failed: CausationID mismatch")
	}
	if restored.SchemaURL() != original.SchemaURL() {
		t.Errorf("Round-trip failed: SchemaURL mismatch")
	}
}

// TestJSONSerializer_BufferPoolReuse tests that buffer pool is being reused
func TestJSONSerializer_BufferPoolReuse(t *testing.T) {
	serializer := NewJSONSerializer()

	event, _ := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)

	// Serialize multiple times to ensure buffer pool is working
	for i := 0; i < 5; i++ {
		_, err := serializer.Serialize(event)
		if err != nil {
			t.Errorf("Serialize() iteration %d error = %v", i, err)
		}
	}
}

// TestProtobufSerializer_Serialize_NotImplemented tests protobuf serialization not implemented
func TestProtobufSerializer_Serialize_NotImplemented(t *testing.T) {
	event, _ := entity.NewEvent(
		"event-1",
		"test-source",
		"test.type",
		"/test",
		[]byte(`{"name": "test"}`),
		"application/json",
	)

	serializer := NewProtobufSerializer()
	_, err := serializer.Serialize(event)

	if err == nil {
		t.Error("Serialize() expected error (not implemented), got nil")
	}
}

// TestProtobufSerializer_Serialize_NilEvent tests protobuf serialization with nil event
func TestProtobufSerializer_Serialize_NilEvent(t *testing.T) {
	serializer := NewProtobufSerializer()
	_, err := serializer.Serialize(nil)

	if err == nil {
		t.Error("Serialize() with nil event expected error, got nil")
	}
}

// TestProtobufSerializer_Deserialize_NotImplemented tests protobuf deserialization not implemented
func TestProtobufSerializer_Deserialize_NotImplemented(t *testing.T) {
	serializer := NewProtobufSerializer()
	_, err := serializer.Deserialize([]byte(`test`))

	if err == nil {
		t.Error("Deserialize() expected error (not implemented), got nil")
	}
}

// TestProtobufSerializer_Deserialize_EmptyData tests protobuf deserialization with empty data
func TestProtobufSerializer_Deserialize_EmptyData(t *testing.T) {
	serializer := NewProtobufSerializer()
	_, err := serializer.Deserialize([]byte{})

	if err == nil {
		t.Error("Deserialize() with empty data expected error, got nil")
	}
}

// TestSerializerFactory_JSON tests factory creates JSON serializer
func TestSerializerFactory_JSON(t *testing.T) {
	serializer, err := SerializerFactory("application/json")

	if err != nil {
		t.Errorf("SerializerFactory() error = %v, want nil", err)
	}

	if _, ok := serializer.(*JSONSerializer); !ok {
		t.Error("SerializerFactory() did not return JSONSerializer for application/json")
	}
}

// TestSerializerFactory_Protobuf tests factory creates Protobuf serializer
func TestSerializerFactory_Protobuf(t *testing.T) {
	tests := []string{
		"application/protobuf",
		"application/octet-stream",
	}

	for _, contentType := range tests {
		t.Run(contentType, func(t *testing.T) {
			serializer, err := SerializerFactory(contentType)

			if err != nil {
				t.Errorf("SerializerFactory() error = %v, want nil", err)
			}

			if _, ok := serializer.(*ProtobufSerializer); !ok {
				t.Error("SerializerFactory() did not return ProtobufSerializer")
			}
		})
	}
}

// TestSerializerFactory_Unknown tests factory defaults to JSON for unknown type
func TestSerializerFactory_Unknown(t *testing.T) {
	serializer, err := SerializerFactory("application/unknown")

	if err != nil {
		t.Errorf("SerializerFactory() error = %v, want nil", err)
	}

	if _, ok := serializer.(*JSONSerializer); !ok {
		t.Error("SerializerFactory() did not default to JSONSerializer for unknown type")
	}
}

// TestSerializerFactory_Empty tests factory with empty content type
func TestSerializerFactory_Empty(t *testing.T) {
	serializer, err := SerializerFactory("")

	if err != nil {
		t.Errorf("SerializerFactory() error = %v, want nil", err)
	}

	if _, ok := serializer.(*JSONSerializer); !ok {
		t.Error("SerializerFactory() did not default to JSONSerializer for empty type")
	}
}
