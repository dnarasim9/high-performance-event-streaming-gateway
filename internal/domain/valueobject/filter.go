package valueobject

import (
	"regexp"
)

// Priority represents the priority level of an event.
// Defined in valueobject to avoid circular dependencies between entity and valueobject.
type Priority int

// Priority constants define the supported priority levels.
const (
	PriorityUnspecified Priority = iota
	PriorityLow
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// String returns the string representation of Priority.
func (p Priority) String() string {
	switch p {
	case PriorityUnspecified:
		return "Unspecified"
	case PriorityLow:
		return "Low"
	case PriorityMedium:
		return "Medium"
	case PriorityHigh:
		return "High"
	case PriorityCritical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// Filterable defines the interface that an event must satisfy to be filtered.
// This avoids circular dependencies between entity and valueobject packages.
type Filterable interface {
	Source() string
	Type() string
	Subject() string
	Priority() Priority
	Metadata() map[string]string
}

// EventFilter is a value object that represents filtering criteria for events.
// It uses regex patterns for flexible matching of event attributes.
// All fields are private to prevent external mutation.
type EventFilter struct {
	sourcePattern   string
	typePattern     string
	subjectPattern  string
	metadataFilters map[string]string
	minPriority     Priority

	// compiled regex patterns for performance optimization
	sourceRegex  *regexp.Regexp
	typeRegex    *regexp.Regexp
	subjectRegex *regexp.Regexp
}

// NewEventFilter creates a new EventFilter and compiles the regex patterns.
// Returns an error if any pattern is invalid regex.
func NewEventFilter(
	sourcePattern string,
	typePattern string,
	subjectPattern string,
	metadataFilters map[string]string,
	minPriority Priority,
) (EventFilter, error) {
	// Create a copy of metadata filters to prevent external mutation
	metadataFiltersCopy := make(map[string]string, len(metadataFilters))
	for k, v := range metadataFilters {
		metadataFiltersCopy[k] = v
	}

	filter := EventFilter{
		sourcePattern:   sourcePattern,
		typePattern:     typePattern,
		subjectPattern:  subjectPattern,
		metadataFilters: metadataFiltersCopy,
		minPriority:     minPriority,
	}

	var err error
	if sourcePattern != "" {
		filter.sourceRegex, err = regexp.Compile(sourcePattern)
		if err != nil {
			return EventFilter{}, err
		}
	}

	if typePattern != "" {
		filter.typeRegex, err = regexp.Compile(typePattern)
		if err != nil {
			return EventFilter{}, err
		}
	}

	if subjectPattern != "" {
		filter.subjectRegex, err = regexp.Compile(subjectPattern)
		if err != nil {
			return EventFilter{}, err
		}
	}

	return filter, nil
}

// SourcePattern returns the source pattern.
func (f EventFilter) SourcePattern() string {
	return f.sourcePattern
}

// TypePattern returns the type pattern.
func (f EventFilter) TypePattern() string {
	return f.typePattern
}

// SubjectPattern returns the subject pattern.
func (f EventFilter) SubjectPattern() string {
	return f.subjectPattern
}

// MetadataFilters returns a copy of the metadata filters map.
// This prevents external code from mutating the internal state.
func (f EventFilter) MetadataFilters() map[string]string {
	filtersCopy := make(map[string]string, len(f.metadataFilters))
	for k, v := range f.metadataFilters {
		filtersCopy[k] = v
	}
	return filtersCopy
}

// MinPriority returns the minimum priority threshold.
func (f EventFilter) MinPriority() Priority {
	return f.minPriority
}

// Matches checks if the given filterable event matches this filter's criteria.
// Returns true only if the event matches all specified filter conditions.
func (f EventFilter) Matches(event Filterable) bool {
	if f.sourceRegex != nil && !f.sourceRegex.MatchString(event.Source()) {
		return false
	}

	if f.typeRegex != nil && !f.typeRegex.MatchString(event.Type()) {
		return false
	}

	if f.subjectRegex != nil && !f.subjectRegex.MatchString(event.Subject()) {
		return false
	}

	if event.Priority() < f.minPriority {
		return false
	}

	if len(f.metadataFilters) > 0 {
		eventMetadata := event.Metadata()
		for key, expectedValue := range f.metadataFilters {
			actualValue, ok := eventMetadata[key]
			if !ok || actualValue != expectedValue {
				return false
			}
		}
	}

	return true
}

// Empty returns true if the filter has no matching criteria set.
// An empty filter will match all events.
func (f EventFilter) Empty() bool {
	return f.sourcePattern == "" &&
		f.typePattern == "" &&
		f.subjectPattern == "" &&
		len(f.metadataFilters) == 0 &&
		f.minPriority == PriorityUnspecified
}

// AddMetadataFilter adds or updates a metadata filter criterion.
// Returns a new EventFilter with the added filter (value semantics).
func (f EventFilter) AddMetadataFilter(key, value string) EventFilter {
	newFilters := make(map[string]string, len(f.metadataFilters)+1)
	for k, v := range f.metadataFilters {
		newFilters[k] = v
	}
	newFilters[key] = value
	f.metadataFilters = newFilters
	return f
}

// RemoveMetadataFilter removes a metadata filter criterion.
// Returns a new EventFilter with the removed filter (value semantics).
func (f EventFilter) RemoveMetadataFilter(key string) EventFilter {
	if len(f.metadataFilters) > 0 {
		newFilters := make(map[string]string, len(f.metadataFilters)-1)
		for k, v := range f.metadataFilters {
			if k != key {
				newFilters[k] = v
			}
		}
		f.metadataFilters = newFilters
	}
	return f
}
