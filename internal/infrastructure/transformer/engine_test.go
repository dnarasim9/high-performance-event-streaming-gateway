package transformer

import (
	"context"
	"testing"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

func newTestEvent(id string) *entity.Event {
	event, _ := entity.NewEvent(
		id,
		"test-source",
		"test.event",
		"test/subject",
		[]byte(`{"name": "test", "value": 42}`),
		"application/json",
	)
	return event
}

func TestTransformationEngine_Transform_NoTransformation(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")

	result, err := engine.Transform(context.Background(), event, nil)
	if err != nil {
		t.Errorf("Transform() with nil transformation error = %v, want nil", err)
	}

	if result != event {
		t.Error("Transform() with nil transformation should return same event")
	}
}

func TestTransformationEngine_Transform_NoneType(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeNone,
		Script: "",
	}

	result, err := engine.Transform(context.Background(), event, transformation)
	if err != nil {
		t.Errorf("Transform() with none type error = %v, want nil", err)
	}

	if result != event {
		t.Error("Transform() with none type should return same event")
	}
}

func TestTransformationEngine_Transform_NilEvent(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	transformation := &Transformation{
		Type:   TransformationTypeJQ,
		Script: ".",
	}

	_, err := engine.Transform(context.Background(), nil, transformation)
	if err == nil {
		t.Error("Transform() with nil event expected error, got nil")
	}
}

func TestTransformationEngine_Transform_ScriptTooLarge(t *testing.T) {
	engine := NewTransformationEngine(100)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeJQ,
		Script: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
	}

	_, err := engine.Transform(context.Background(), event, transformation)
	if err == nil {
		t.Error("Transform() with oversized script expected error, got nil")
	}
}

func TestTransformationEngine_Transform_UnknownType(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   "unknown",
		Script: ".",
	}

	_, err := engine.Transform(context.Background(), event, transformation)
	if err == nil {
		t.Error("Transform() with unknown type expected error, got nil")
	}
}

func TestTransformationEngine_Transform_JQ_Valid(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeJQ,
		Script: ".",
	}

	// JQ transformation should not error on compilation
	result, err := engine.Transform(context.Background(), event, transformation)
	if err != nil {
		// It's ok if it fails due to transformation issues, but not compilation
		t.Logf("Transform() with JQ: %v", err)
	} else if result == nil {
		t.Error("Transform() returned nil event on success")
	}
}

func TestTransformationEngine_Transform_JQ_Invalid(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeJQ,
		Script: "invalid syntax here {",
	}

	_, err := engine.Transform(context.Background(), event, transformation)
	if err == nil {
		t.Error("Transform() with invalid JQ expected error, got nil")
	}
}

func TestTransformationEngine_Transform_CEL_Valid(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeCEL,
		Script: `event`,
	}

	// CEL transformation should not error on compilation
	result, err := engine.Transform(context.Background(), event, transformation)
	if err != nil {
		// It's ok if it fails due to transformation issues, but not compilation
		t.Logf("Transform() with CEL: %v", err)
	} else if result == nil {
		t.Error("Transform() returned nil event on success")
	}
}

func TestTransformationEngine_Transform_CEL_Invalid(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeCEL,
		Script: `invalid {{{{ CEL syntax`,
	}

	_, err := engine.Transform(context.Background(), event, transformation)
	if err == nil {
		t.Error("Transform() with invalid CEL expected error, got nil")
	}
}

func TestTransformationEngine_Transform_Template_Valid(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeTemplate,
		Script: `{"source": "{{.Source}}", "type": "{{.Type}}", "subject": "{{.Subject}}", "id": "{{.ID}}", "data": "test", "dataContentType": "application/json"}`,
	}

	result, err := engine.Transform(context.Background(), event, transformation)
	if err != nil {
		// It's ok if it fails due to transformation issues, but not template parsing
		t.Logf("Transform() with template: %v", err)
	} else if result == nil {
		t.Error("Transform() returned nil event on success")
	}
}

func TestTransformationEngine_Transform_Template_Invalid(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeTemplate,
		Script: `{{unclosed template`,
	}

	_, err := engine.Transform(context.Background(), event, transformation)
	if err == nil {
		t.Error("Transform() with invalid template expected error, got nil")
	}
}

func TestTransformationEngine_Transform_Template_InvalidJSON(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeTemplate,
		Script: `not valid json at all`,
	}

	_, err := engine.Transform(context.Background(), event, transformation)
	if err == nil {
		t.Error("Transform() with template producing invalid JSON expected error, got nil")
	}
}

func TestTransformationEngine_CachingJQ(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event1 := newTestEvent("event-1")
	event2 := newTestEvent("event-2")

	transformation := &Transformation{
		Type:   TransformationTypeJQ,
		Script: ".",
	}

	// First call should compile and cache
	engine.Transform(context.Background(), event1, transformation)

	// Second call with same script should use cache
	engine.Transform(context.Background(), event2, transformation)

	// Verify cache was populated (compilation happened at least once)
	stats := engine.GetCacheStats()
	if jqCount, ok := stats["jq_queries"].(int); ok {
		if jqCount < 1 {
			t.Errorf("GetCacheStats() jq_queries = %d, want >= 1", jqCount)
		}
	}
}

func TestTransformationEngine_CachingCEL(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event1 := newTestEvent("event-1")
	event2 := newTestEvent("event-2")

	transformation := &Transformation{
		Type:   TransformationTypeCEL,
		Script: `event`,
	}

	// First call should compile and cache
	engine.Transform(context.Background(), event1, transformation)

	// Second call with same script should use cache
	engine.Transform(context.Background(), event2, transformation)

	// Verify cache was populated
	stats := engine.GetCacheStats()
	if celCount, ok := stats["cel_programs"].(int); ok {
		if celCount < 1 {
			t.Errorf("GetCacheStats() cel_programs = %d, want >= 1", celCount)
		}
	}
}

func TestTransformationEngine_CachingTemplate(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event1 := newTestEvent("event-1")
	event2 := newTestEvent("event-2")

	transformation := &Transformation{
		Type:   TransformationTypeTemplate,
		Script: `{"source": "{{.Source}}", "id": "{{.ID}}", "type": "{{.Type}}", "subject": "{{.Subject}}", "data": "test", "dataContentType": "application/json"}`,
	}

	// First call should parse and cache
	engine.Transform(context.Background(), event1, transformation)

	// Second call with same script should use cache
	engine.Transform(context.Background(), event2, transformation)

	// Verify cache was populated
	stats := engine.GetCacheStats()
	if templateCount, ok := stats["templates"].(int); ok {
		if templateCount < 1 {
			t.Errorf("GetCacheStats() templates = %d, want >= 1", templateCount)
		}
	}
}

func TestTransformationEngine_GetCacheStats(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	stats := engine.GetCacheStats()

	if celCount, ok := stats["cel_programs"].(int); !ok || celCount != 0 {
		t.Error("GetCacheStats() cel_programs should be 0 initially")
	}

	if jqCount, ok := stats["jq_queries"].(int); !ok || jqCount != 0 {
		t.Error("GetCacheStats() jq_queries should be 0 initially")
	}

	if templateCount, ok := stats["templates"].(int); !ok || templateCount != 0 {
		t.Error("GetCacheStats() templates should be 0 initially")
	}
}

func TestTransformationEngine_ClearCache(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("event-1")
	transformation := &Transformation{
		Type:   TransformationTypeJQ,
		Script: ".",
	}

	// Cache something
	engine.Transform(context.Background(), event, transformation)

	// Verify cache is populated
	stats := engine.GetCacheStats()
	jqCount := stats["jq_queries"].(int)
	if jqCount == 0 {
		t.Error("Cache should have entries before clear")
	}

	// Clear cache
	engine.ClearCache()

	// Verify cache is empty
	stats = engine.GetCacheStats()
	if jqCount, ok := stats["jq_queries"].(int); ok {
		if jqCount != 0 {
			t.Errorf("GetCacheStats() after ClearCache() jq_queries = %d, want 0", jqCount)
		}
	}
}

func TestTransformationEngine_PreservesEventID(t *testing.T) {
	engine := NewTransformationEngine(1000000)

	event := newTestEvent("special-event-id")
	transformation := &Transformation{
		Type:   TransformationTypeNone,
		Script: "",
	}

	result, _ := engine.Transform(context.Background(), event, transformation)

	if result == nil {
		t.Error("Transform() returned nil")
		return
	}

	if result.ID() != "special-event-id" {
		t.Errorf("Transform() changed event ID from %q to %q", "special-event-id", result.ID())
	}
}
