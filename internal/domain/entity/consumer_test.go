package entity

import (
	"testing"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/valueobject"
)

func TestNewConsumer_Success(t *testing.T) {
	id := "consumer-123"
	name := "test-consumer"
	groupID := "group-456"
	filter, _ := valueobject.NewEventFilter(".*", ".*", ".*", nil, valueobject.PriorityLow)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")

	consumer, err := NewConsumer(id, name, groupID, filter, transformation)

	if err != nil {
		t.Errorf("NewConsumer() got error %v, want nil", err)
	}

	if consumer == nil {
		t.Fatal("NewConsumer() returned nil")
	}

	if consumer.ID() != id {
		t.Errorf("ID() = %q, want %q", consumer.ID(), id)
	}
	if consumer.Name() != name {
		t.Errorf("Name() = %q, want %q", consumer.Name(), name)
	}
	if consumer.GroupID() != groupID {
		t.Errorf("GroupID() = %q, want %q", consumer.GroupID(), groupID)
	}

	// Verify default status
	if consumer.Status() != ConsumerStatusInactive {
		t.Errorf("Status() = %v, want %v", consumer.Status(), ConsumerStatusInactive)
	}

	// Verify timestamps are set
	if consumer.CreatedAt().IsZero() {
		t.Error("CreatedAt() is zero")
	}
	if consumer.UpdatedAt().IsZero() {
		t.Error("UpdatedAt() is zero")
	}
}

func TestNewConsumer_EmptyID(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")

	_, err := NewConsumer("", "name", "group", filter, transformation)

	if err == nil {
		t.Error("NewConsumer() with empty ID should return error")
	}
}

func TestNewConsumer_EmptyName(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")

	_, err := NewConsumer("id", "", "group", filter, transformation)

	if err == nil {
		t.Error("NewConsumer() with empty name should return error")
	}
}

func TestNewConsumer_EmptyGroupID(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")

	_, err := NewConsumer("id", "name", "", filter, transformation)

	if err == nil {
		t.Error("NewConsumer() with empty groupID should return error")
	}
}

func TestNewConsumer_InvalidTransformation(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	// Create invalid transformation (non-None type with empty expression)
	invalidTransformation := valueobject.Transformation{
		Type:       valueobject.TransformationTypeJQ,
		Expression: "",
	}

	_, err := NewConsumer("id", "name", "group", filter, invalidTransformation)

	if err == nil {
		t.Error("NewConsumer() with invalid transformation should return error")
	}
}

func TestConsumer_Activate(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	if consumer.Status() != ConsumerStatusInactive {
		t.Fatalf("Initial status should be Inactive, got %v", consumer.Status())
	}

	err := consumer.Activate()
	if err != nil {
		t.Errorf("Activate() returned error %v", err)
	}

	if consumer.Status() != ConsumerStatusActive {
		t.Errorf("Status() = %v, want %v", consumer.Status(), ConsumerStatusActive)
	}

	if !consumer.IsActive() {
		t.Error("IsActive() should return true")
	}
}

func TestConsumer_Deactivate(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	consumer.Activate()

	err := consumer.Deactivate()
	if err != nil {
		t.Errorf("Deactivate() returned error %v", err)
	}

	if consumer.Status() != ConsumerStatusInactive {
		t.Errorf("Status() = %v, want %v", consumer.Status(), ConsumerStatusInactive)
	}

	if !consumer.IsInactive() {
		t.Error("IsInactive() should return true")
	}
}

func TestConsumer_Drain(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	consumer.Drain()

	if consumer.Status() != ConsumerStatusDraining {
		t.Errorf("Status() = %v, want %v", consumer.Status(), ConsumerStatusDraining)
	}

	if !consumer.IsDraining() {
		t.Error("IsDraining() should return true")
	}
}

func TestConsumer_Activate_FromDraining(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	consumer.Drain()
	err := consumer.Activate()

	if err == nil {
		t.Error("Activate() on draining consumer should return error")
	}
}

func TestConsumer_Deactivate_FromDraining(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	consumer.Drain()
	err := consumer.Deactivate()

	if err == nil {
		t.Error("Deactivate() on draining consumer should return error")
	}
}

func TestConsumer_UpdateFilter(t *testing.T) {
	oldFilter, _ := valueobject.NewEventFilter("old.*", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", oldFilter, transformation)

	consumer.Activate()

	newFilter, _ := valueobject.NewEventFilter("new.*", "", "", nil, valueobject.PriorityUnspecified)
	err := consumer.UpdateFilter(newFilter)

	if err != nil {
		t.Errorf("UpdateFilter() returned error %v", err)
	}

	if consumer.Filter().SourcePattern() != "new.*" {
		t.Errorf("Filter not updated correctly")
	}
}

func TestConsumer_UpdateFilter_OnDrainingConsumer(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	consumer.Drain()

	newFilter, _ := valueobject.NewEventFilter("new.*", "", "", nil, valueobject.PriorityUnspecified)
	err := consumer.UpdateFilter(newFilter)

	if err == nil {
		t.Error("UpdateFilter() on draining consumer should return error")
	}
}

func TestConsumer_StatusTransitions(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(*Consumer)
		operation      func(*Consumer) error
		expectedStatus ConsumerStatus
		wantErr        bool
	}{
		{
			name: "Inactive_to_Active",
			setup: func(c *Consumer) {
				// Starts as inactive
			},
			operation: func(c *Consumer) error {
				return c.Activate()
			},
			expectedStatus: ConsumerStatusActive,
			wantErr:        false,
		},
		{
			name: "Active_to_Inactive",
			setup: func(c *Consumer) {
				c.Activate()
			},
			operation: func(c *Consumer) error {
				return c.Deactivate()
			},
			expectedStatus: ConsumerStatusInactive,
			wantErr:        false,
		},
		{
			name: "Active_to_Draining",
			setup: func(c *Consumer) {
				c.Activate()
			},
			operation: func(c *Consumer) error {
				c.Drain()
				return nil
			},
			expectedStatus: ConsumerStatusDraining,
			wantErr:        false,
		},
		{
			name: "Inactive_to_Draining",
			setup: func(c *Consumer) {
				// Starts as inactive
			},
			operation: func(c *Consumer) error {
				c.Drain()
				return nil
			},
			expectedStatus: ConsumerStatusDraining,
			wantErr:        false,
		},
		{
			name: "Draining_to_Active_Error",
			setup: func(c *Consumer) {
				c.Drain()
			},
			operation: func(c *Consumer) error {
				return c.Activate()
			},
			expectedStatus: ConsumerStatusDraining,
			wantErr:        true,
		},
		{
			name: "Draining_to_Inactive_Error",
			setup: func(c *Consumer) {
				c.Drain()
			},
			operation: func(c *Consumer) error {
				return c.Deactivate()
			},
			expectedStatus: ConsumerStatusDraining,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
			transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
			consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

			tt.setup(consumer)
			err := tt.operation(consumer)

			if (err != nil) != tt.wantErr {
				t.Errorf("operation() error = %v, wantErr %v", err, tt.wantErr)
			}

			if consumer.Status() != tt.expectedStatus {
				t.Errorf("Status() = %v, want %v", consumer.Status(), tt.expectedStatus)
			}
		})
	}
}

func TestConsumer_UpdatedAt_Changes(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	initialUpdatedAt := consumer.UpdatedAt()
	time.Sleep(10 * time.Millisecond)

	consumer.Activate()
	afterActivateUpdatedAt := consumer.UpdatedAt()

	if !afterActivateUpdatedAt.After(initialUpdatedAt) {
		t.Error("UpdatedAt should be updated after Activate()")
	}

	time.Sleep(10 * time.Millisecond)
	consumer.Deactivate()
	afterDeactivateUpdatedAt := consumer.UpdatedAt()

	if !afterDeactivateUpdatedAt.After(afterActivateUpdatedAt) {
		t.Error("UpdatedAt should be updated after Deactivate()")
	}
}

func TestConsumer_MatchesEvent(t *testing.T) {
	// Create a filter that matches events from "test-source"
	filter, _ := valueobject.NewEventFilter("test-source", ".*", ".*", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	// Create matching event
	matchingEvent, _ := NewEvent("evt-1", "test-source", "type", "subject", []byte("data"), "application/json")
	if !consumer.MatchesEvent(matchingEvent) {
		t.Error("MatchesEvent() should return true for matching event")
	}

	// Create non-matching event
	nonMatchingEvent, _ := NewEvent("evt-2", "other-source", "type", "subject", []byte("data"), "application/json")
	if consumer.MatchesEvent(nonMatchingEvent) {
		t.Error("MatchesEvent() should return false for non-matching event")
	}
}

func TestConsumer_Filter(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("source.*", "type.*", "subject.*", nil, valueobject.PriorityLow)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeNone, "")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	consumerFilter := consumer.Filter()

	if consumerFilter.SourcePattern() != "source.*" {
		t.Errorf("SourcePattern() = %q, want %q", consumerFilter.SourcePattern(), "source.*")
	}
	if consumerFilter.TypePattern() != "type.*" {
		t.Errorf("TypePattern() = %q, want %q", consumerFilter.TypePattern(), "type.*")
	}
	if consumerFilter.SubjectPattern() != "subject.*" {
		t.Errorf("SubjectPattern() = %q, want %q", consumerFilter.SubjectPattern(), "subject.*")
	}
	if consumerFilter.MinPriority() != valueobject.PriorityLow {
		t.Errorf("MinPriority() = %v, want %v", consumerFilter.MinPriority(), valueobject.PriorityLow)
	}
}

func TestConsumer_Transformation(t *testing.T) {
	filter, _ := valueobject.NewEventFilter("", "", "", nil, valueobject.PriorityUnspecified)
	transformation, _ := valueobject.NewTransformation(valueobject.TransformationTypeJQ, ".field")
	consumer, _ := NewConsumer("id", "name", "group", filter, transformation)

	consumerTransformation := consumer.Transformation()

	if consumerTransformation.Type != valueobject.TransformationTypeJQ {
		t.Errorf("Type = %v, want %v", consumerTransformation.Type, valueobject.TransformationTypeJQ)
	}
	if consumerTransformation.Expression != ".field" {
		t.Errorf("Expression = %q, want %q", consumerTransformation.Expression, ".field")
	}
}

func TestConsumerStatus_String(t *testing.T) {
	tests := []struct {
		status   ConsumerStatus
		expected string
	}{
		{ConsumerStatusActive, "Active"},
		{ConsumerStatusInactive, "Inactive"},
		{ConsumerStatusDraining, "Draining"},
		{ConsumerStatus(999), "Unknown"},
	}

	for _, tt := range tests {
		if tt.status.String() != tt.expected {
			t.Errorf("String() = %q, want %q", tt.status.String(), tt.expected)
		}
	}
}
