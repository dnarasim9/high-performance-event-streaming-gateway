package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// SchemaValidator validates data against registered JSON schemas.
type SchemaValidator struct { //nolint:revive // established API
	schemas  map[string]*jsonschema.Schema
	cache    map[string]time.Time
	cacheTTL time.Duration
	mu       sync.RWMutex
}

// NewSchemaValidator creates a new SchemaValidator.
func NewSchemaValidator(cacheTTL time.Duration) *SchemaValidator {
	return &SchemaValidator{
		schemas:  make(map[string]*jsonschema.Schema),
		cache:    make(map[string]time.Time),
		cacheTTL: cacheTTL,
	}
}

// RegisterSchema registers a JSON schema by ID.
func (sv *SchemaValidator) RegisterSchema(schemaID string, schemaJSON []byte) error {
	if schemaID == "" {
		return fmt.Errorf("schema ID cannot be empty")
	}

	if len(schemaJSON) == 0 {
		return fmt.Errorf("schema JSON cannot be empty")
	}

	// Compile the schema
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(string(schemaJSON))
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()

	sv.schemas[schemaID] = schema
	sv.cache[schemaID] = time.Now()

	return nil
}

// GetSchema retrieves a registered schema by ID.
func (sv *SchemaValidator) GetSchema(schemaID string) (*jsonschema.Schema, error) {
	if schemaID == "" {
		return nil, fmt.Errorf("schema ID cannot be empty")
	}

	sv.mu.RLock()
	schema, exists := sv.schemas[schemaID]
	cacheTime, cacheExists := sv.cache[schemaID]
	sv.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("schema %q not found", schemaID)
	}

	// Check if cache expired
	if cacheExists && time.Since(cacheTime) > sv.cacheTTL {
		sv.mu.Lock()
		delete(sv.schemas, schemaID)
		delete(sv.cache, schemaID)
		sv.mu.Unlock()
		return nil, fmt.Errorf("schema %q cache expired", schemaID)
	}

	return schema, nil
}

// Validate validates data against a registered schema.
func (sv *SchemaValidator) Validate(ctx context.Context, schemaID string, data []byte) error {
	if schemaID == "" {
		return fmt.Errorf("schema ID cannot be empty")
	}

	if len(data) == 0 {
		return fmt.Errorf("data cannot be empty")
	}

	// Get the schema
	schema, err := sv.GetSchema(schemaID)
	if err != nil {
		return err
	}

	// Parse JSON data
	var jsonData interface{}
	if err := parseJSON(data, &jsonData); err != nil {
		return fmt.Errorf("failed to parse JSON data: %w", err)
	}

	// Validate the data
	if err := schema.Validate(jsonData); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}

// parseJSON parses JSON bytes into a value
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// UnregisterSchema removes a schema by ID.
func (sv *SchemaValidator) UnregisterSchema(schemaID string) error {
	if schemaID == "" {
		return fmt.Errorf("schema ID cannot be empty")
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()

	if _, exists := sv.schemas[schemaID]; !exists {
		return fmt.Errorf("schema %q not found", schemaID)
	}

	delete(sv.schemas, schemaID)
	delete(sv.cache, schemaID)

	return nil
}

// ListSchemas returns a list of registered schema IDs.
func (sv *SchemaValidator) ListSchemas() []string {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	ids := make([]string, 0, len(sv.schemas))
	for id := range sv.schemas {
		ids = append(ids, id)
	}

	return ids
}

// ClearCache clears expired schemas from cache.
func (sv *SchemaValidator) ClearCache() {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	now := time.Now()
	for id, cacheTime := range sv.cache {
		if now.Sub(cacheTime) > sv.cacheTTL {
			delete(sv.schemas, id)
			delete(sv.cache, id)
		}
	}
}

// GetStats returns statistics about the validator.
func (sv *SchemaValidator) GetStats() map[string]interface{} {
	sv.mu.RLock()
	defer sv.mu.RUnlock()

	return map[string]interface{}{
		"total_schemas": len(sv.schemas),
		"cache_ttl":     sv.cacheTTL.String(),
	}
}
