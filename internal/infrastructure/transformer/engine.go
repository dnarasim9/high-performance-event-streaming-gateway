package transformer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/google/cel-go/cel"
	"github.com/itchyny/gojq"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// TransformationType represents the type of transformation.
type TransformationType string

// TransformationTypeJQ, TransformationTypeCEL, TransformationTypeTemplate, and TransformationTypeNone
// are the supported transformation types.
const (
	TransformationTypeJQ       TransformationType = "jq"
	TransformationTypeCEL      TransformationType = "cel"
	TransformationTypeTemplate TransformationType = "template"
	TransformationTypeNone     TransformationType = "none"
)

// Transformation represents a transformation to apply to an event.
type Transformation struct {
	Type   TransformationType `json:"type"`
	Script string             `json:"script"`
}

// TransformationEngine applies transformations to events.
type TransformationEngine struct {
	maxScriptSize int
	celPrograms   map[string]cel.Program
	jqQueries     map[string]*gojq.Query
	templates     map[string]*template.Template
}

// NewTransformationEngine creates a new TransformationEngine.
func NewTransformationEngine(maxScriptSize int) *TransformationEngine {
	return &TransformationEngine{
		maxScriptSize: maxScriptSize,
		celPrograms:   make(map[string]cel.Program),
		jqQueries:     make(map[string]*gojq.Query),
		templates:     make(map[string]*template.Template),
	}
}

// Transform applies a transformation to an event.
func (te *TransformationEngine) Transform(
	ctx context.Context,
	event *entity.Event,
	transformation *Transformation,
) (*entity.Event, error) {
	if event == nil {
		return nil, fmt.Errorf("event cannot be nil")
	}

	if transformation == nil || transformation.Type == TransformationTypeNone {
		// No transformation
		return event, nil
	}

	if len(transformation.Script) > te.maxScriptSize {
		return nil, fmt.Errorf("transformation script exceeds maximum size of %d bytes", te.maxScriptSize)
	}

	switch transformation.Type {
	case TransformationTypeJQ:
		return te.transformJQ(ctx, event, transformation.Script)
	case TransformationTypeCEL:
		return te.transformCEL(ctx, event, transformation.Script)
	case TransformationTypeTemplate:
		return te.transformTemplate(ctx, event, transformation.Script)
	case TransformationTypeNone:
		// No transformation
		return event, nil
	default:
		return nil, fmt.Errorf("unknown transformation type: %s", transformation.Type)
	}
}

// transformJQ applies a JQ transformation to an event.
func (te *TransformationEngine) transformJQ(_ctx context.Context, event *entity.Event, script string) (*entity.Event, error) {
	// Get or compile the JQ query
	var query *gojq.Query
	if q, exists := te.jqQueries[script]; exists {
		query = q
	} else {
		var err error
		query, err = gojq.Parse(script)
		if err != nil {
			return nil, fmt.Errorf("failed to parse JQ script: %w", err)
		}
		te.jqQueries[script] = query
	}

	eventMap, err := te.eventToMap(event)
	if err != nil {
		return nil, err
	}

	// Apply JQ transformation
	iter := query.Run(eventMap)
	results := make([]interface{}, 0)

	for {
		v, ok := iter.Next()
		if !ok {
			break
		}

		if err, ok := v.(error); ok {
			return nil, fmt.Errorf("JQ execution error: %w", err)
		}

		results = append(results, v)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("JQ transformation produced no results")
	}

	return te.buildTransformedEvent(event, results[0])
}

// transformCEL applies a CEL transformation to an event.
func (te *TransformationEngine) transformCEL(_ctx context.Context, event *entity.Event, script string) (*entity.Event, error) {
	// Get or compile the CEL program
	var program cel.Program
	if p, exists := te.celPrograms[script]; exists {
		program = p
	} else {
		p, err := te.compileCELProgram(script)
		if err != nil {
			return nil, err
		}
		program = p
		te.celPrograms[script] = p
	}

	eventMap, err := te.eventToMap(event)
	if err != nil {
		return nil, err
	}

	// Execute CEL program
	result, _, err := program.Eval(map[string]interface{}{
		"event": eventMap,
	})
	if err != nil {
		return nil, fmt.Errorf("CEL execution error: %w", err)
	}

	return te.buildTransformedEvent(event, result.Value())
}

// compileCELProgram compiles a CEL script into a program.
func (te *TransformationEngine) compileCELProgram(script string) (cel.Program, error) {
	env, err := cel.NewEnv(
		cel.Variable("event", cel.MapType(cel.StringType, cel.DynType)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	ast, issues := env.Compile(script)
	if issues.Err() != nil {
		return nil, fmt.Errorf("failed to compile CEL script: %w", issues.Err())
	}

	program, err := env.Program(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL program: %w", err)
	}

	return program, nil
}

// eventToMap converts an event to a map for transformation.
func (te *TransformationEngine) eventToMap(event *entity.Event) (map[string]interface{}, error) {
	eventData, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event: %w", err)
	}

	var eventMap map[string]interface{}
	if err := json.Unmarshal(eventData, &eventMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return eventMap, nil
}

// buildTransformedEvent builds a new event from transformation result.
func (te *TransformationEngine) buildTransformedEvent(event *entity.Event, resultValue interface{}) (*entity.Event, error) {
	resultData, err := json.Marshal(resultValue)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transformation result: %w", err)
	}

	var transformedEventData map[string]interface{}
	if err := json.Unmarshal(resultData, &transformedEventData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transformed event: %w", err)
	}

	// Preserve original ID and timestamp if not in transformation result
	if id, ok := transformedEventData["id"]; !ok || id == "" {
		transformedEventData["id"] = event.ID()
	}
	if ts, ok := transformedEventData["timestamp"]; !ok || ts == "" {
		transformedEventData["timestamp"] = event.Timestamp().String()
	}

	updatedData, err := json.Marshal(transformedEventData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated event: %w", err)
	}

	var dto struct {
		ID              string            `json:"id"`
		Source          string            `json:"source"`
		Type            string            `json:"type"`
		Subject         string            `json:"subject"`
		Data            []byte            `json:"data"`
		DataContentType string            `json:"dataContentType"`
		SchemaURL       string            `json:"schemaUrl"`
		Timestamp       string            `json:"timestamp"`
		Metadata        map[string]string `json:"metadata"`
		CorrelationID   string            `json:"correlationId"`
		CausationID     string            `json:"causationId"`
		PartitionKey    string            `json:"partitionKey"`
		Priority        int               `json:"priority"`
	}
	if err := json.Unmarshal(updatedData, &dto); err != nil {
		return nil, fmt.Errorf("failed to parse transformed event: %w", err)
	}

	priority := entity.Priority(dto.Priority)
	newEvent, err := entity.NewEvent(
		dto.ID,
		dto.Source,
		dto.Type,
		dto.Subject,
		dto.Data,
		dto.DataContentType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transformed event: %w", err)
	}

	newEvent = newEvent.
		WithSchemaURL(dto.SchemaURL).
		WithMetadata(dto.Metadata).
		WithCorrelation(dto.CorrelationID, dto.CausationID).
		WithPartitionKey(dto.PartitionKey).
		WithPriority(priority)

	return newEvent, nil
}

// transformTemplate applies a template transformation to an event.
func (te *TransformationEngine) transformTemplate(_ctx context.Context, event *entity.Event, script string) (*entity.Event, error) {
	// Get or parse the template
	var tmpl *template.Template
	if t, exists := te.templates[script]; exists {
		tmpl = t
	} else {
		var err error
		tmpl, err = template.New("transform").Parse(script)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}
		te.templates[script] = tmpl
	}

	// Execute template
	var buf strings.Builder
	if err := tmpl.Execute(&buf, event); err != nil {
		return nil, fmt.Errorf("template execution error: %w", err)
	}

	// Parse result as JSON
	var transformedEventData map[string]interface{}
	if err := json.Unmarshal([]byte(buf.String()), &transformedEventData); err != nil {
		return nil, fmt.Errorf("failed to parse template result as JSON: %w", err)
	}

	// Preserve original ID and timestamp if not in transformation result
	if id, ok := transformedEventData["id"]; !ok || id == "" {
		transformedEventData["id"] = event.ID()
	}
	if ts, ok := transformedEventData["timestamp"]; !ok || ts == "" {
		transformedEventData["timestamp"] = event.Timestamp().String()
	}

	// Re-marshal and parse back into DTO structure
	updatedData, err := json.Marshal(transformedEventData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal updated event: %w", err)
	}

	var dto struct {
		ID              string            `json:"id"`
		Source          string            `json:"source"`
		Type            string            `json:"type"`
		Subject         string            `json:"subject"`
		Data            []byte            `json:"data"`
		DataContentType string            `json:"dataContentType"`
		SchemaURL       string            `json:"schemaUrl"`
		Timestamp       string            `json:"timestamp"`
		Metadata        map[string]string `json:"metadata"`
		CorrelationID   string            `json:"correlationId"`
		CausationID     string            `json:"causationId"`
		PartitionKey    string            `json:"partitionKey"`
		Priority        int               `json:"priority"`
	}
	if err := json.Unmarshal(updatedData, &dto); err != nil {
		return nil, fmt.Errorf("failed to parse transformed event: %w", err)
	}

	// Create a new event using the domain constructor
	priority := entity.Priority(dto.Priority)
	newEvent, err := entity.NewEvent(
		dto.ID,
		dto.Source,
		dto.Type,
		dto.Subject,
		dto.Data,
		dto.DataContentType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transformed event: %w", err)
	}

	// Apply optional fields using builder pattern
	newEvent = newEvent.
		WithSchemaURL(dto.SchemaURL).
		WithMetadata(dto.Metadata).
		WithCorrelation(dto.CorrelationID, dto.CausationID).
		WithPartitionKey(dto.PartitionKey).
		WithPriority(priority)

	return newEvent, nil
}

// ClearCache clears the transformation cache.
func (te *TransformationEngine) ClearCache() {
	te.celPrograms = make(map[string]cel.Program)
	te.jqQueries = make(map[string]*gojq.Query)
	te.templates = make(map[string]*template.Template)
}

// GetCacheStats returns statistics about the transformation cache.
func (te *TransformationEngine) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"cel_programs": len(te.celPrograms),
		"jq_queries":   len(te.jqQueries),
		"templates":    len(te.templates),
	}
}
