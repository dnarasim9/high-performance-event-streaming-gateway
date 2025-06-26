package entity

import (
	"bytes"
	"testing"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

func TestNewEvent_Success(t *testing.T) {
	eventID := "event-123"
	source := "test-source"
	eventType := "test.event"
	subject := "test/subject"
	data := []byte(`{"key": "value"}`)
	contentType := "application/json"

	event, err := NewEvent(eventID, source, eventType, subject, data, contentType)

	if err != nil {
		t.Errorf("NewEvent() got error %v, want nil", err)
	}

	if event == nil {
		t.Fatal("NewEvent() returned nil event")
	}

	if event.ID() != eventID {
		t.Errorf("ID() = %q, want %q", event.ID(), eventID)
	}
	if event.Source() != source {
		t.Errorf("Source() = %q, want %q", event.Source(), source)
	}
	if event.Type() != eventType {
		t.Errorf("Type() = %q, want %q", event.Type(), eventType)
	}
	if event.Subject() != subject {
		t.Errorf("Subject() = %q, want %q", event.Subject(), subject)
	}
	if !bytes.Equal(event.Data(), data) {
		t.Errorf("Data() = %v, want %v", event.Data(), data)
	}
	if event.DataContentType() != contentType {
		t.Errorf("DataContentType() = %q, want %q", event.DataContentType(), contentType)
	}

	// Verify default values
	if event.Priority() != PriorityMedium {
		t.Errorf("Priority() = %v, want %v", event.Priority(), PriorityMedium)
	}
	if event.CorrelationID() != "" {
		t.Errorf("CorrelationID() = %q, want empty", event.CorrelationID())
	}
	if len(event.Metadata()) != 0 {
		t.Errorf("Metadata() = %v, want empty map", event.Metadata())
	}
}

func TestNewEvent_EmptyID(t *testing.T) {
	_, err := NewEvent("", "source", "type", "subject", []byte("data"), "application/json")

	if err == nil {
		t.Error("NewEvent() with empty ID should return error")
	}
	if err.Error() != "event ID cannot be empty" {
		t.Errorf("NewEvent() error = %q, want %q", err.Error(), "event ID cannot be empty")
	}
}

func TestNewEvent_EmptySource(t *testing.T) {
	_, err := NewEvent("id", "", "type", "subject", []byte("data"), "application/json")

	if err == nil {
		t.Error("NewEvent() with empty source should return error")
	}
	if err.Error() != "event source cannot be empty" {
		t.Errorf("NewEvent() error = %q, want %q", err.Error(), "event source cannot be empty")
	}
}

func TestNewEvent_EmptyType(t *testing.T) {
	_, err := NewEvent("id", "source", "", "subject", []byte("data"), "application/json")

	if err == nil {
		t.Error("NewEvent() with empty type should return error")
	}
	if err.Error() != "event type cannot be empty" {
		t.Errorf("NewEvent() error = %q, want %q", err.Error(), "event type cannot be empty")
	}
}

func TestNewEvent_EmptyData(t *testing.T) {
	_, err := NewEvent("id", "source", "type", "subject", []byte{}, "application/json")

	if err == nil {
		t.Error("NewEvent() with empty data should return error")
	}
	if err.Error() != "event data cannot be empty" {
		t.Errorf("NewEvent() error = %q, want %q", err.Error(), "event data cannot be empty")
	}
}

func TestNewEvent_EmptyContentType(t *testing.T) {
	_, err := NewEvent("id", "source", "type", "subject", []byte("data"), "")

	if err == nil {
		t.Error("NewEvent() with empty content type should return error")
	}
	if err.Error() != "event data content type cannot be empty" {
		t.Errorf("NewEvent() error = %q, want %q", err.Error(), "event data content type cannot be empty")
	}
}

func TestEvent_WithMetadata(t *testing.T) {
	event, err := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")
	if err != nil {
		t.Fatalf("NewEvent() failed: %v", err)
	}

	metadata := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	newEvent := event.WithMetadata(metadata)

	// Original event should be unchanged
	if len(event.Metadata()) != 0 {
		t.Errorf("Original event metadata modified")
	}

	// New event should have metadata
	if len(newEvent.Metadata()) != 2 {
		t.Errorf("WithMetadata() returned event with %d metadata items, want 2", len(newEvent.Metadata()))
	}

	for key, value := range metadata {
		if newEvent.Metadata()[key] != value {
			t.Errorf("Metadata[%q] = %q, want %q", key, newEvent.Metadata()[key], value)
		}
	}
}

func TestEvent_WithMetadata_Merge(t *testing.T) {
	event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")
	event = event.WithMetadata(map[string]string{"key1": "value1"})

	// Add more metadata
	event = event.WithMetadata(map[string]string{"key2": "value2"})

	metadata := event.Metadata()
	if len(metadata) != 2 {
		t.Errorf("WithMetadata() merging failed: got %d items, want 2", len(metadata))
	}
	if metadata["key1"] != "value1" || metadata["key2"] != "value2" {
		t.Errorf("WithMetadata() merge failed: %v", metadata)
	}
}

func TestEvent_WithCorrelation(t *testing.T) {
	event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")

	correlationID := "corr-123"
	causationID := "cause-456"

	newEvent := event.WithCorrelation(correlationID, causationID)

	// Original event should be unchanged
	if event.CorrelationID() != "" {
		t.Error("Original event correlation ID modified")
	}

	// New event should have IDs
	if newEvent.CorrelationID() != correlationID {
		t.Errorf("CorrelationID() = %q, want %q", newEvent.CorrelationID(), correlationID)
	}
	if newEvent.CausationID() != causationID {
		t.Errorf("CausationID() = %q, want %q", newEvent.CausationID(), causationID)
	}
}

func TestEvent_WithPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority valueobject.Priority
	}{
		{"Low", valueobject.PriorityLow},
		{"Medium", valueobject.PriorityMedium},
		{"High", valueobject.PriorityHigh},
		{"Critical", valueobject.PriorityCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")

			newEvent := event.WithPriority(tt.priority)

			// Original event should be unchanged
			if event.Priority() != PriorityMedium {
				t.Error("Original event priority modified")
			}

			// New event should have priority
			if newEvent.Priority() != tt.priority {
				t.Errorf("Priority() = %v, want %v", newEvent.Priority(), tt.priority)
			}
		})
	}
}

func TestEvent_IsHighPriority(t *testing.T) {
	tests := []struct {
		name         string
		priority     valueobject.Priority
		wantHighPrio bool
	}{
		{"Low", valueobject.PriorityLow, false},
		{"Medium", valueobject.PriorityMedium, false},
		{"High", valueobject.PriorityHigh, true},
		{"Critical", valueobject.PriorityCritical, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")
			event = event.WithPriority(tt.priority)

			if event.IsHighPriority() != tt.wantHighPrio {
				t.Errorf("IsHighPriority() = %v, want %v", event.IsHighPriority(), tt.wantHighPrio)
			}
		})
	}
}

func TestEvent_Immutability(t *testing.T) {
	originalMetadata := map[string]string{
		"key1": "value1",
	}

	event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")
	event = event.WithMetadata(originalMetadata)

	// Modify the original metadata map
	originalMetadata["key1"] = "modified"
	originalMetadata["key2"] = "new"

	// Event's metadata should not be affected
	eventMetadata := event.Metadata()
	if eventMetadata["key1"] != "value1" {
		t.Errorf("Event metadata was modified externally: %v", eventMetadata)
	}
	if _, ok := eventMetadata["key2"]; ok {
		t.Error("New key added to original metadata affects event")
	}
}

func TestEvent_DataCopy(t *testing.T) {
	originalData := []byte("original data")
	event, _ := NewEvent("id", "source", "type", "subject", originalData, "application/json")

	// Get a copy and modify it
	dataCopy := event.DataCopy()
	dataCopy[0] = 'X'

	// Original event's data should be unchanged
	if event.Data()[0] != 'o' {
		t.Error("Event data was modified when using DataCopy()")
	}

	// Get data again to ensure it's still original
	data := event.Data()
	if !bytes.Equal(data, originalData) {
		t.Errorf("Event data() = %s, want %s", string(data), string(originalData))
	}
}

func TestEvent_Validate(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		source      string
		eventType   string
		data        []byte
		contentType string
		wantErr     bool
	}{
		{"Valid", "id", "source", "type", []byte("data"), "application/json", false},
		{"EmptyID", "", "source", "type", []byte("data"), "application/json", true},
		{"EmptySource", "id", "", "type", []byte("data"), "application/json", true},
		{"EmptyType", "id", "source", "", []byte("data"), "application/json", true},
		{"EmptyData", "id", "source", "type", []byte{}, "application/json", true},
		{"EmptyContentType", "id", "source", "type", []byte("data"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				id:              tt.id,
				source:          tt.source,
				eventType:       tt.eventType,
				data:            tt.data,
				dataContentType: tt.contentType,
			}

			err := event.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEvent_Age(t *testing.T) {
	event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")

	time.Sleep(10 * time.Millisecond)

	age := event.Age()
	if age < 10*time.Millisecond {
		t.Errorf("Age() = %v, want >= 10ms", age)
	}
}

func TestEvent_WithPartitionKey(t *testing.T) {
	event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")

	partitionKey := "partition-key-123"
	newEvent := event.WithPartitionKey(partitionKey)

	if event.PartitionKey() != "" {
		t.Error("Original event partition key modified")
	}
	if newEvent.PartitionKey() != partitionKey {
		t.Errorf("PartitionKey() = %q, want %q", newEvent.PartitionKey(), partitionKey)
	}
}

func TestEvent_WithSchemaURL(t *testing.T) {
	event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")

	schemaURL := "https://schema.example.com/schema.json"
	newEvent := event.WithSchemaURL(schemaURL)

	if event.SchemaURL() != "" {
		t.Error("Original event schema URL modified")
	}
	if newEvent.SchemaURL() != schemaURL {
		t.Errorf("SchemaURL() = %q, want %q", newEvent.SchemaURL(), schemaURL)
	}
}

func TestEvent_ChainableBuilder(t *testing.T) {
	event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")

	// Chain multiple operations
	finalEvent := event.
		WithMetadata(map[string]string{"key": "value"}).
		WithPriority(PriorityHigh).
		WithCorrelation("corr-id", "cause-id").
		WithPartitionKey("partition").
		WithSchemaURL("https://schema.example.com")

	// Verify all properties are set correctly
	if finalEvent.Metadata()["key"] != "value" {
		t.Error("Metadata not set correctly in chain")
	}
	if finalEvent.Priority() != PriorityHigh {
		t.Error("Priority not set correctly in chain")
	}
	if finalEvent.CorrelationID() != "corr-id" {
		t.Error("CorrelationID not set correctly in chain")
	}
	if finalEvent.PartitionKey() != "partition" {
		t.Error("PartitionKey not set correctly in chain")
	}
	if finalEvent.SchemaURL() != "https://schema.example.com" {
		t.Error("SchemaURL not set correctly in chain")
	}

	// Verify original event is unchanged
	if len(event.Metadata()) != 0 {
		t.Error("Original event metadata modified")
	}
	if event.Priority() != PriorityMedium {
		t.Error("Original event priority modified")
	}
}

func TestEvent_Timestamp(t *testing.T) {
	before := time.Now().UTC()
	event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")
	after := time.Now().UTC()

	timestamp := event.Timestamp()

	if timestamp.Before(before) || timestamp.After(after.Add(1*time.Second)) {
		t.Errorf("Timestamp() = %v, want between %v and %v", timestamp, before, after)
	}
}

func TestEvent_CreatedAt(t *testing.T) {
	before := time.Now().UTC()
	event, _ := NewEvent("id", "source", "type", "subject", []byte("data"), "application/json")
	after := time.Now().UTC()

	createdAt := event.CreatedAt()

	if createdAt.Before(before) || createdAt.After(after.Add(1*time.Second)) {
		t.Errorf("CreatedAt() = %v, want between %v and %v", createdAt, before, after)
	}
}
