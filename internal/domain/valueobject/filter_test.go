package valueobject

import (
	"testing"
)

type mockEvent struct {
	source   string
	typ      string
	subject  string
	priority Priority
	metadata map[string]string
}

func (me mockEvent) Source() string              { return me.source }
func (me mockEvent) Type() string                { return me.typ }
func (me mockEvent) Subject() string             { return me.subject }
func (me mockEvent) Priority() Priority          { return me.priority }
func (me mockEvent) Metadata() map[string]string { return me.metadata }

func TestNewEventFilter_Valid(t *testing.T) {
	filter, err := NewEventFilter("source.*", "type.*", "subject.*", nil, PriorityMedium)

	if err != nil {
		t.Errorf("NewEventFilter() got error %v, want nil", err)
	}

	if filter.SourcePattern() != "source.*" {
		t.Errorf("SourcePattern() = %q, want %q", filter.SourcePattern(), "source.*")
	}
	if filter.TypePattern() != "type.*" {
		t.Errorf("TypePattern() = %q, want %q", filter.TypePattern(), "type.*")
	}
	if filter.SubjectPattern() != "subject.*" {
		t.Errorf("SubjectPattern() = %q, want %q", filter.SubjectPattern(), "subject.*")
	}
	if filter.MinPriority() != PriorityMedium {
		t.Errorf("MinPriority() = %v, want %v", filter.MinPriority(), PriorityMedium)
	}
}

func TestNewEventFilter_InvalidSourceRegex(t *testing.T) {
	_, err := NewEventFilter("[invalid(regex", "", "", nil, PriorityUnspecified)

	if err == nil {
		t.Error("NewEventFilter() with invalid source regex should return error")
	}
}

func TestNewEventFilter_InvalidTypeRegex(t *testing.T) {
	_, err := NewEventFilter("", "[invalid(regex", "", nil, PriorityUnspecified)

	if err == nil {
		t.Error("NewEventFilter() with invalid type regex should return error")
	}
}

func TestNewEventFilter_InvalidSubjectRegex(t *testing.T) {
	_, err := NewEventFilter("", "", "[invalid(regex", nil, PriorityUnspecified)

	if err == nil {
		t.Error("NewEventFilter() with invalid subject regex should return error")
	}
}

func TestNewEventFilter_EmptyPatterns(t *testing.T) {
	filter, err := NewEventFilter("", "", "", nil, PriorityUnspecified)

	if err != nil {
		t.Errorf("NewEventFilter() with empty patterns got error %v, want nil", err)
	}

	if !filter.Empty() {
		t.Error("Filter with empty patterns should report Empty() = true")
	}
}

func TestEventFilter_Matches_Source(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		source    string
		wantMatch bool
	}{
		{"Exact match", "test-source", "test-source", true},
		{"Regex match", "test.*", "test-source", true},
		{"Regex mismatch", "test.*", "prod-source", false},
		{"Case sensitive", "Test.*", "test-source", false},
		{"Dotall pattern", ".*source", "test-source", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, _ := NewEventFilter(tt.pattern, "", "", nil, PriorityUnspecified)
			event := mockEvent{source: tt.source, priority: PriorityMedium}

			if filter.Matches(event) != tt.wantMatch {
				t.Errorf("Matches() = %v, want %v", filter.Matches(event), tt.wantMatch)
			}
		})
	}
}

func TestEventFilter_Matches_Type(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		typ       string
		wantMatch bool
	}{
		{"Exact match", "user.created", "user.created", true},
		{"Regex match", "user\\..*", "user.created", true},
		{"Regex mismatch", "order\\..*", "user.created", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, _ := NewEventFilter("", tt.pattern, "", nil, PriorityUnspecified)
			event := mockEvent{typ: tt.typ, priority: PriorityMedium}

			if filter.Matches(event) != tt.wantMatch {
				t.Errorf("Matches() = %v, want %v", filter.Matches(event), tt.wantMatch)
			}
		})
	}
}

func TestEventFilter_Matches_Subject(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		subject   string
		wantMatch bool
	}{
		{"Exact match", "/users/123", "/users/123", true},
		{"Regex match", "/users/.*", "/users/123", true},
		{"Regex mismatch", "/orders/.*", "/users/123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, _ := NewEventFilter("", "", tt.pattern, nil, PriorityUnspecified)
			event := mockEvent{subject: tt.subject, priority: PriorityMedium}

			if filter.Matches(event) != tt.wantMatch {
				t.Errorf("Matches() = %v, want %v", filter.Matches(event), tt.wantMatch)
			}
		})
	}
}

func TestEventFilter_Matches_Priority(t *testing.T) {
	tests := []struct {
		name        string
		minPriority Priority
		eventPrio   Priority
		wantMatch   bool
	}{
		{"Equal priority", PriorityMedium, PriorityMedium, true},
		{"Higher event priority", PriorityMedium, PriorityHigh, true},
		{"Lower event priority", PriorityMedium, PriorityLow, false},
		{"Unspecified filter", PriorityUnspecified, PriorityLow, true},
		{"Critical meets High filter", PriorityHigh, PriorityCritical, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, _ := NewEventFilter("", "", "", nil, tt.minPriority)
			event := mockEvent{priority: tt.eventPrio}

			if filter.Matches(event) != tt.wantMatch {
				t.Errorf("Matches() = %v, want %v", filter.Matches(event), tt.wantMatch)
			}
		})
	}
}

func TestEventFilter_Matches_Metadata(t *testing.T) {
	tests := []struct {
		name            string
		metadataFilters map[string]string
		eventMetadata   map[string]string
		wantMatch       bool
	}{
		{
			"Exact metadata match",
			map[string]string{"env": "prod"},
			map[string]string{"env": "prod"},
			true,
		},
		{
			"Metadata mismatch value",
			map[string]string{"env": "prod"},
			map[string]string{"env": "dev"},
			false,
		},
		{
			"Metadata missing key",
			map[string]string{"env": "prod"},
			map[string]string{"region": "us-east"},
			false,
		},
		{
			"Multiple metadata filters all match",
			map[string]string{"env": "prod", "region": "us-east"},
			map[string]string{"env": "prod", "region": "us-east", "extra": "value"},
			true,
		},
		{
			"Multiple metadata filters partial match",
			map[string]string{"env": "prod", "region": "us-west"},
			map[string]string{"env": "prod", "region": "us-east"},
			false,
		},
		{
			"No metadata filters",
			nil,
			map[string]string{"env": "prod"},
			true,
		},
		{
			"Empty metadata filters",
			map[string]string{},
			map[string]string{"env": "prod"},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, _ := NewEventFilter("", "", "", tt.metadataFilters, PriorityUnspecified)
			event := mockEvent{metadata: tt.eventMetadata}

			if filter.Matches(event) != tt.wantMatch {
				t.Errorf("Matches() = %v, want %v", filter.Matches(event), tt.wantMatch)
			}
		})
	}
}

func TestEventFilter_Matches_Combined(t *testing.T) {
	// Test filter that matches all criteria
	filter, _ := NewEventFilter(
		"app\\..*",
		"user\\..*",
		"/users/.*",
		map[string]string{"env": "prod"},
		PriorityHigh,
	)

	// Matching event
	matchingEvent := mockEvent{
		source:   "app.service",
		typ:      "user.created",
		subject:  "/users/123",
		priority: PriorityCritical,
		metadata: map[string]string{"env": "prod", "region": "us-east"},
	}

	if !filter.Matches(matchingEvent) {
		t.Error("Matches() should return true for event matching all criteria")
	}

	// Event failing source pattern
	failSourceEvent := mockEvent{
		source:   "api.service",
		typ:      "user.created",
		subject:  "/users/123",
		priority: PriorityCritical,
		metadata: map[string]string{"env": "prod"},
	}

	if filter.Matches(failSourceEvent) {
		t.Error("Matches() should return false when source doesn't match")
	}

	// Event failing priority
	failPriorityEvent := mockEvent{
		source:   "app.service",
		typ:      "user.created",
		subject:  "/users/123",
		priority: PriorityMedium,
		metadata: map[string]string{"env": "prod"},
	}

	if filter.Matches(failPriorityEvent) {
		t.Error("Matches() should return false when priority is too low")
	}

	// Event failing metadata
	failMetadataEvent := mockEvent{
		source:   "app.service",
		typ:      "user.created",
		subject:  "/users/123",
		priority: PriorityCritical,
		metadata: map[string]string{"env": "dev"},
	}

	if filter.Matches(failMetadataEvent) {
		t.Error("Matches() should return false when metadata doesn't match")
	}
}

func TestEventFilter_Empty(t *testing.T) {
	tests := []struct {
		name      string
		filter    func() EventFilter
		wantEmpty bool
	}{
		{
			"Empty filter",
			func() EventFilter {
				f, _ := NewEventFilter("", "", "", nil, PriorityUnspecified)
				return f
			},
			true,
		},
		{
			"Filter with source pattern",
			func() EventFilter {
				f, _ := NewEventFilter("source.*", "", "", nil, PriorityUnspecified)
				return f
			},
			false,
		},
		{
			"Filter with type pattern",
			func() EventFilter {
				f, _ := NewEventFilter("", "type.*", "", nil, PriorityUnspecified)
				return f
			},
			false,
		},
		{
			"Filter with subject pattern",
			func() EventFilter {
				f, _ := NewEventFilter("", "", "subject.*", nil, PriorityUnspecified)
				return f
			},
			false,
		},
		{
			"Filter with metadata",
			func() EventFilter {
				f, _ := NewEventFilter("", "", "", map[string]string{"key": "value"}, PriorityUnspecified)
				return f
			},
			false,
		},
		{
			"Filter with priority",
			func() EventFilter {
				f, _ := NewEventFilter("", "", "", nil, PriorityHigh)
				return f
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := tt.filter()
			if filter.Empty() != tt.wantEmpty {
				t.Errorf("Empty() = %v, want %v", filter.Empty(), tt.wantEmpty)
			}
		})
	}
}

func TestEventFilter_AddMetadataFilter(t *testing.T) {
	filter, _ := NewEventFilter("", "", "", nil, PriorityUnspecified)

	// Add first metadata filter
	filter = filter.AddMetadataFilter("env", "prod")

	if len(filter.MetadataFilters()) != 1 {
		t.Errorf("MetadataFilters() length = %d, want 1", len(filter.MetadataFilters()))
	}
	if filter.MetadataFilters()["env"] != "prod" {
		t.Errorf("MetadataFilters()[env] = %q, want %q", filter.MetadataFilters()["env"], "prod")
	}

	// Add second metadata filter
	filter = filter.AddMetadataFilter("region", "us-east")

	if len(filter.MetadataFilters()) != 2 {
		t.Errorf("MetadataFilters() length = %d, want 2", len(filter.MetadataFilters()))
	}
	if filter.MetadataFilters()["region"] != "us-east" {
		t.Errorf("MetadataFilters()[region] = %q, want %q", filter.MetadataFilters()["region"], "us-east")
	}

	// Update existing filter
	filter = filter.AddMetadataFilter("env", "dev")

	if len(filter.MetadataFilters()) != 2 {
		t.Errorf("MetadataFilters() length = %d, want 2", len(filter.MetadataFilters()))
	}
	if filter.MetadataFilters()["env"] != "dev" {
		t.Errorf("MetadataFilters()[env] = %q, want %q", filter.MetadataFilters()["env"], "dev")
	}
}

func TestEventFilter_RemoveMetadataFilter(t *testing.T) {
	metadata := map[string]string{
		"env":    "prod",
		"region": "us-east",
	}

	filter, _ := NewEventFilter("", "", "", metadata, PriorityUnspecified)

	// Remove one filter
	filter = filter.RemoveMetadataFilter("region")

	if len(filter.MetadataFilters()) != 1 {
		t.Errorf("MetadataFilters() length = %d, want 1", len(filter.MetadataFilters()))
	}
	if _, ok := filter.MetadataFilters()["region"]; ok {
		t.Error("region key should be removed from MetadataFilters()")
	}
	if filter.MetadataFilters()["env"] != "prod" {
		t.Error("env key should still be in MetadataFilters()")
	}

	// Remove another filter
	filter = filter.RemoveMetadataFilter("env")

	if len(filter.MetadataFilters()) != 0 {
		t.Errorf("MetadataFilters() length = %d, want 0", len(filter.MetadataFilters()))
	}

	// Remove non-existent filter (should not error)
	filter = filter.RemoveMetadataFilter("nonexistent")

	if len(filter.MetadataFilters()) != 0 {
		t.Errorf("MetadataFilters() length = %d, want 0", len(filter.MetadataFilters()))
	}
}

func TestEventFilter_RemoveMetadataFilter_NonExistent(t *testing.T) {
	metadata := map[string]string{
		"env": "prod",
	}

	filter, _ := NewEventFilter("", "", "", metadata, PriorityUnspecified)

	// Try to remove non-existent key
	originalLen := len(filter.MetadataFilters())
	filter = filter.RemoveMetadataFilter("nonexistent")

	if len(filter.MetadataFilters()) != originalLen {
		t.Error("Removing non-existent key should not change filter")
	}
}

func TestEventFilter_Chaining(t *testing.T) {
	filter, _ := NewEventFilter("source.*", "type.*", "subject.*", nil, PriorityUnspecified)

	// Chain metadata operations
	filter = filter.
		AddMetadataFilter("env", "prod").
		AddMetadataFilter("region", "us-east").
		AddMetadataFilter("team", "backend")

	if len(filter.MetadataFilters()) != 3 {
		t.Errorf("MetadataFilters() length = %d, want 3", len(filter.MetadataFilters()))
	}

	// Chain removal
	filter = filter.RemoveMetadataFilter("team")

	if len(filter.MetadataFilters()) != 2 {
		t.Errorf("MetadataFilters() length = %d, want 2", len(filter.MetadataFilters()))
	}

	// Verify original patterns still exist
	if filter.SourcePattern() != "source.*" {
		t.Error("SourcePattern() was modified")
	}
}

func TestEventFilter_Matches_EmptyFilter(t *testing.T) {
	// Empty filter should match any event
	filter, _ := NewEventFilter("", "", "", nil, PriorityUnspecified)
	event := mockEvent{
		source:   "any-source",
		typ:      "any.type",
		subject:  "any-subject",
		priority: PriorityLow,
		metadata: map[string]string{"any": "metadata"},
	}

	if !filter.Matches(event) {
		t.Error("Empty filter should match any event")
	}
}

func TestEventFilter_Matches_NoRegexPatterns(t *testing.T) {
	// Filter with only priority and metadata criteria
	filter, _ := NewEventFilter("", "", "", map[string]string{"env": "prod"}, PriorityHigh)

	matchingEvent := mockEvent{
		source:   "any-source",
		typ:      "any.type",
		subject:  "any-subject",
		priority: PriorityCritical,
		metadata: map[string]string{"env": "prod"},
	}

	nonMatchingEvent := mockEvent{
		source:   "any-source",
		typ:      "any.type",
		subject:  "any-subject",
		priority: PriorityLow,
		metadata: map[string]string{"env": "prod"},
	}

	if !filter.Matches(matchingEvent) {
		t.Error("Filter should match event with matching priority and metadata")
	}

	if filter.Matches(nonMatchingEvent) {
		t.Error("Filter should not match event with mismatched priority")
	}
}
