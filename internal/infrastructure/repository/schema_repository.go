package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// InMemorySchemaRepository is a simple in-memory implementation of SchemaRepository.
// It is thread-safe for concurrent access.
type InMemorySchemaRepository struct {
	sync.RWMutex
	schemas map[string]*entity.Schema
}

// NewInMemorySchemaRepository creates a new InMemorySchemaRepository.
func NewInMemorySchemaRepository() *InMemorySchemaRepository {
	return &InMemorySchemaRepository{
		schemas: make(map[string]*entity.Schema),
	}
}

// Save stores or updates a schema in the repository.
func (r *InMemorySchemaRepository) Save(ctx context.Context, schema *entity.Schema) error {
	r.Lock()
	defer r.Unlock()
	if r.schemas == nil {
		r.schemas = make(map[string]*entity.Schema)
	}
	r.schemas[schema.ID()] = schema
	return nil
}

// Get retrieves a schema by its ID.
func (r *InMemorySchemaRepository) Get(ctx context.Context, id string) (*entity.Schema, error) {
	r.RLock()
	defer r.RUnlock()
	if schema, exists := r.schemas[id]; exists {
		return schema, nil
	}
	return nil, fmt.Errorf("schema not found")
}

// GetByNameVersion retrieves a schema by its name and version.
func (r *InMemorySchemaRepository) GetByNameVersion(ctx context.Context, name, version string) (*entity.Schema, error) {
	r.RLock()
	defer r.RUnlock()
	for _, schema := range r.schemas {
		if schema.Name() == name && schema.Version() == version {
			return schema, nil
		}
	}
	return nil, fmt.Errorf("schema not found")
}

// List retrieves all schemas, optionally filtered by name.
func (r *InMemorySchemaRepository) List(ctx context.Context, name string) ([]*entity.Schema, error) {
	r.RLock()
	defer r.RUnlock()
	var result []*entity.Schema
	for _, schema := range r.schemas {
		if name == "" || schema.Name() == name {
			result = append(result, schema)
		}
	}
	return result, nil
}

// Delete removes a schema from the repository.
func (r *InMemorySchemaRepository) Delete(ctx context.Context, id string) error {
	r.Lock()
	defer r.Unlock()
	delete(r.schemas, id)
	return nil
}
