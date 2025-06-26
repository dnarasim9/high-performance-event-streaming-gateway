package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// Mock SchemaRepository
type MockSchemaRepository struct {
	SaveFunc             func(ctx context.Context, schema *entity.Schema) error
	GetFunc              func(ctx context.Context, id string) (*entity.Schema, error)
	GetByNameVersionFunc func(ctx context.Context, name, version string) (*entity.Schema, error)
	ListFunc             func(ctx context.Context, name string) ([]*entity.Schema, error)
	DeleteFunc           func(ctx context.Context, id string) error
}

func (m *MockSchemaRepository) Save(ctx context.Context, schema *entity.Schema) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, schema)
	}
	return nil
}

func (m *MockSchemaRepository) Get(ctx context.Context, id string) (*entity.Schema, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockSchemaRepository) GetByNameVersion(ctx context.Context, name, version string) (*entity.Schema, error) {
	if m.GetByNameVersionFunc != nil {
		return m.GetByNameVersionFunc(ctx, name, version)
	}
	return nil, nil
}

func (m *MockSchemaRepository) List(ctx context.Context, name string) ([]*entity.Schema, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, name)
	}
	return nil, nil
}

func (m *MockSchemaRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func TestSchemaService_Register_Success(t *testing.T) {
	tests := []struct {
		name string
		req  *dto.RegisterSchemaRequest
	}{
		{
			name: "json schema",
			req: &dto.RegisterSchemaRequest{
				Name:       "user",
				Version:    "1.0.0",
				Format:     "JSONSchema",
				Definition: `{"type": "object"}`,
			},
		},
		{
			name: "avro schema",
			req: &dto.RegisterSchemaRequest{
				Name:       "event",
				Version:    "2.0.0",
				Format:     "Avro",
				Definition: `{"type": "record", "name": "Event"}`,
			},
		},
		{
			name: "protobuf schema",
			req: &dto.RegisterSchemaRequest{
				Name:       "message",
				Version:    "1.5.0",
				Format:     "Protobuf",
				Definition: `syntax = "proto3";`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSchemaRepository{}
			validator := &MockSchemaValidator{}
			metrics := &MockMetricsCollector{}

			service := NewSchemaService(repo, validator, metrics, newTestLogger())

			schemaID, err := service.Register(context.Background(), tt.req)
			if err != nil {
				t.Errorf("Register() error = %v, want nil", err)
			}

			if schemaID == "" {
				t.Error("Register() returned empty schema ID")
			}
		})
	}
}

func TestSchemaService_Register_InvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *dto.RegisterSchemaRequest
	}{
		{
			name: "empty name",
			req: &dto.RegisterSchemaRequest{
				Name:       "",
				Version:    "1.0.0",
				Format:     "JSONSchema",
				Definition: `{}`,
			},
		},
		{
			name: "empty version",
			req: &dto.RegisterSchemaRequest{
				Name:       "schema",
				Version:    "",
				Format:     "JSONSchema",
				Definition: `{}`,
			},
		},
		{
			name: "empty definition",
			req: &dto.RegisterSchemaRequest{
				Name:       "schema",
				Version:    "1.0.0",
				Format:     "JSONSchema",
				Definition: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockSchemaRepository{}
			validator := &MockSchemaValidator{}
			metrics := &MockMetricsCollector{}

			service := NewSchemaService(repo, validator, metrics, newTestLogger())

			_, err := service.Register(context.Background(), tt.req)
			if err == nil {
				t.Error("Register() expected error, got nil")
			}
		})
	}
}

func TestSchemaService_Register_DuplicateSchema(t *testing.T) {
	req := &dto.RegisterSchemaRequest{
		Name:       "user",
		Version:    "1.0.0",
		Format:     "JSONSchema",
		Definition: `{}`,
	}

	existingSchema, _ := entity.NewSchema("existing-id", "user", "1.0.0", entity.SchemaFormatJSONSchema, `{}`)

	repo := &MockSchemaRepository{
		GetByNameVersionFunc: func(ctx context.Context, name, version string) (*entity.Schema, error) {
			return existingSchema, nil
		},
	}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewSchemaService(repo, validator, metrics, newTestLogger())

	_, err := service.Register(context.Background(), req)
	if err == nil {
		t.Error("Register() expected error for duplicate schema, got nil")
	}
}

func TestSchemaService_Register_RepositoryError(t *testing.T) {
	req := &dto.RegisterSchemaRequest{
		Name:       "user",
		Version:    "1.0.0",
		Format:     "JSONSchema",
		Definition: `{}`,
	}

	repo := &MockSchemaRepository{
		SaveFunc: func(ctx context.Context, schema *entity.Schema) error {
			return errors.New("database error")
		},
	}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewSchemaService(repo, validator, metrics, newTestLogger())

	_, err := service.Register(context.Background(), req)
	if err == nil {
		t.Error("Register() expected error, got nil")
	}
}

func TestSchemaService_Get_Success(t *testing.T) {
	schemaID := "schema-123"
	schema, _ := entity.NewSchema(schemaID, "user", "1.0.0", entity.SchemaFormatJSONSchema, `{}`)

	repo := &MockSchemaRepository{
		GetFunc: func(ctx context.Context, id string) (*entity.Schema, error) {
			if id == schemaID {
				return schema, nil
			}
			return nil, errors.New("not found")
		},
	}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewSchemaService(repo, validator, metrics, newTestLogger())

	result, err := service.Get(context.Background(), schemaID)
	if err != nil {
		t.Errorf("Get() error = %v, want nil", err)
	}

	if result == nil {
		t.Error("Get() returned nil schema")
	}

	if result.ID != schemaID {
		t.Errorf("Get() ID = %q, want %q", result.ID, schemaID)
	}
	if result.Name != "user" {
		t.Errorf("Get() Name = %q, want %q", result.Name, "user")
	}
}

func TestSchemaService_Get_NotFound(t *testing.T) {
	repo := &MockSchemaRepository{
		GetFunc: func(ctx context.Context, id string) (*entity.Schema, error) {
			return nil, errors.New("not found")
		},
	}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewSchemaService(repo, validator, metrics, newTestLogger())

	_, err := service.Get(context.Background(), "non-existent")
	if err == nil {
		t.Error("Get() expected error, got nil")
	}
}

func TestSchemaService_List_Success(t *testing.T) {
	schemas := []*entity.Schema{
		func() *entity.Schema {
			s, _ := entity.NewSchema("id1", "user", "1.0.0", entity.SchemaFormatJSONSchema, `{}`)
			return s
		}(),
		func() *entity.Schema {
			s, _ := entity.NewSchema("id2", "user", "2.0.0", entity.SchemaFormatJSONSchema, `{}`)
			return s
		}(),
	}

	repo := &MockSchemaRepository{
		ListFunc: func(ctx context.Context, name string) ([]*entity.Schema, error) {
			if name == "user" {
				return schemas, nil
			}
			return nil, errors.New("not found")
		},
	}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewSchemaService(repo, validator, metrics, newTestLogger())

	results, err := service.List(context.Background(), "user")
	if err != nil {
		t.Errorf("List() error = %v, want nil", err)
	}

	if len(results) != len(schemas) {
		t.Errorf("List() returned %d schemas, want %d", len(results), len(schemas))
	}
}

func TestSchemaService_List_NotFound(t *testing.T) {
	repo := &MockSchemaRepository{
		ListFunc: func(ctx context.Context, name string) ([]*entity.Schema, error) {
			return nil, errors.New("not found")
		},
	}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewSchemaService(repo, validator, metrics, newTestLogger())

	_, err := service.List(context.Background(), "non-existent")
	if err == nil {
		t.Error("List() expected error, got nil")
	}
}

func TestSchemaService_Validate_Success(t *testing.T) {
	schemaID := "schema-123"
	schema, _ := entity.NewSchema(schemaID, "user", "1.0.0", entity.SchemaFormatJSONSchema, `{}`)

	repo := &MockSchemaRepository{
		GetFunc: func(ctx context.Context, id string) (*entity.Schema, error) {
			if id == schemaID {
				return schema, nil
			}
			return nil, errors.New("not found")
		},
	}
	validator := &MockSchemaValidator{
		ValidateFunc: func(ctx context.Context, schemaID string, data []byte) error {
			return nil
		},
	}
	metrics := &MockMetricsCollector{}

	service := NewSchemaService(repo, validator, metrics, newTestLogger())

	req := &dto.ValidateRequest{
		SchemaID: schemaID,
		Data:     []byte(`{"name": "John"}`),
	}

	result, err := service.Validate(context.Background(), req)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}

	if !result.Valid {
		t.Error("Validate() returned invalid result")
	}
}

func TestSchemaService_Validate_SchemaNotFound(t *testing.T) {
	repo := &MockSchemaRepository{
		GetFunc: func(ctx context.Context, id string) (*entity.Schema, error) {
			return nil, errors.New("schema not found")
		},
	}
	validator := &MockSchemaValidator{}
	metrics := &MockMetricsCollector{}

	service := NewSchemaService(repo, validator, metrics, newTestLogger())

	req := &dto.ValidateRequest{
		SchemaID: "non-existent",
		Data:     []byte(`{}`),
	}

	result, err := service.Validate(context.Background(), req)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}

	if result.Valid {
		t.Error("Validate() should return invalid for missing schema")
	}
}

func TestSchemaService_Validate_ValidationFailure(t *testing.T) {
	schemaID := "schema-123"
	schema, _ := entity.NewSchema(schemaID, "user", "1.0.0", entity.SchemaFormatJSONSchema, `{}`)

	repo := &MockSchemaRepository{
		GetFunc: func(ctx context.Context, id string) (*entity.Schema, error) {
			return schema, nil
		},
	}
	validator := &MockSchemaValidator{
		ValidateFunc: func(ctx context.Context, schemaID string, data []byte) error {
			return errors.New("validation failed")
		},
	}
	metrics := &MockMetricsCollector{}

	service := NewSchemaService(repo, validator, metrics, newTestLogger())

	req := &dto.ValidateRequest{
		SchemaID: schemaID,
		Data:     []byte(`{}`),
	}

	result, err := service.Validate(context.Background(), req)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}

	if result.Valid {
		t.Error("Validate() should return invalid when validation fails")
	}
}
