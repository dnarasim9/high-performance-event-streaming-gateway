package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/dto"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/application/port"
	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// SchemaRepository is an interface for schema persistence operations.
type SchemaRepository interface {
	Save(ctx context.Context, schema *entity.Schema) error
	Get(ctx context.Context, id string) (*entity.Schema, error)
	GetByNameVersion(ctx context.Context, name, version string) (*entity.Schema, error)
	List(ctx context.Context, name string) ([]*entity.Schema, error)
	Delete(ctx context.Context, id string) error
}

// SchemaService implements the SchemaManagementUseCase interface.
// It manages schema lifecycle including registration, retrieval, validation, and querying.
type SchemaService struct {
	schemaRepository port.SchemaRepository
	schemaValidator  port.SchemaValidator
	metricsCollector port.MetricsCollector
	logger           *slog.Logger
}

// NewSchemaService creates a new SchemaService with the specified dependencies.
func NewSchemaService(
	schemaRepository port.SchemaRepository,
	schemaValidator port.SchemaValidator,
	metricsCollector port.MetricsCollector,
	logger *slog.Logger,
) *SchemaService {
	return &SchemaService{
		schemaRepository: schemaRepository,
		schemaValidator:  schemaValidator,
		metricsCollector: metricsCollector,
		logger:           logger,
	}
}

// Register registers a new schema in the system.
// Returns the ID of the registered schema or an error.
func (s *SchemaService) Register(ctx context.Context, request *dto.RegisterSchemaRequest) (string, error) {
	startTime := time.Now()
	schemaID := generateID()

	s.logger.InfoContext(
		ctx,
		"Registering new schema",
		slog.String("schemaID", schemaID),
		slog.String("name", request.Name),
		slog.String("version", request.Version),
		slog.String("format", request.Format),
	)

	// Create schema entity
	schema, err := request.ToEntity(schemaID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to create schema entity",
			slog.String("schemaID", schemaID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Register", "entity_creation")
		return "", fmt.Errorf("invalid schema configuration: %w", err)
	}

	// Check if schema with this name and version already exists
	existingSchema, err := s.schemaRepository.GetByNameVersion(ctx, request.Name, request.Version)
	if err == nil && existingSchema != nil {
		s.logger.WarnContext(
			ctx,
			"Schema with same name and version already exists",
			slog.String("schemaID", schemaID),
			slog.String("name", request.Name),
			slog.String("version", request.Version),
		)
		s.metricsCollector.RecordError("Register", "duplicate")
		return "", fmt.Errorf("schema %s v%s already exists", request.Name, request.Version)
	}

	// Save schema to repository
	if err := s.schemaRepository.Save(ctx, schema); err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to save schema to repository",
			slog.String("schemaID", schemaID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Register", "storage")
		return "", fmt.Errorf("failed to register schema: %w", err)
	}

	s.logger.InfoContext(
		ctx,
		"Schema registered successfully",
		slog.String("schemaID", schemaID),
		slog.String("name", request.Name),
		slog.String("version", request.Version),
		slog.String("format", request.Format),
		slog.Duration("duration", time.Since(startTime)),
	)

	// Record metrics
	s.metricsCollector.RecordLatency("register", float64(time.Since(startTime).Milliseconds()))

	return schemaID, nil
}

// Get retrieves a schema by its ID.
// Returns the schema details or an error.
func (s *SchemaService) Get(ctx context.Context, schemaID string) (*dto.SchemaResponse, error) {
	s.logger.InfoContext(
		ctx,
		"Retrieving schema",
		slog.String("schemaID", schemaID),
	)

	schema, err := s.schemaRepository.Get(ctx, schemaID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to retrieve schema",
			slog.String("schemaID", schemaID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Get", "retrieval")
		return nil, fmt.Errorf("schema not found: %w", err)
	}

	return dto.FromEntitySchema(schema), nil
}

// Validate validates event data against a specific schema.
// Returns an error if the data does not conform to the schema.
func (s *SchemaService) Validate(ctx context.Context, request *dto.ValidateRequest) (*dto.ValidateResponse, error) {
	startTime := time.Now()

	s.logger.InfoContext(
		ctx,
		"Validating data against schema",
		slog.String("schemaID", request.SchemaID),
		slog.Int("dataSize", len(request.Data)),
	)

	// Retrieve the schema to verify it exists
	_, err := s.schemaRepository.Get(ctx, request.SchemaID)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to retrieve schema for validation",
			slog.String("schemaID", request.SchemaID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("Validate", "schema_retrieval")
		return &dto.ValidateResponse{
			Valid:    false,
			Error:    fmt.Sprintf("schema not found: %v", err),
			SchemaID: request.SchemaID,
		}, nil
	}

	// Validate the data against the schema
	if err := s.schemaValidator.Validate(ctx, request.SchemaID, request.Data); err != nil {
		s.logger.ErrorContext(
			ctx,
			"Validation failed",
			slog.String("schemaID", request.SchemaID),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordLatency("validate", float64(time.Since(startTime).Milliseconds()))
		s.metricsCollector.RecordError("Validate", "validation")
		return &dto.ValidateResponse{
			Valid:    false,
			Error:    fmt.Sprintf("validation failed: %v", err),
			SchemaID: request.SchemaID,
		}, nil
	}

	s.logger.InfoContext(
		ctx,
		"Data validation successful",
		slog.String("schemaID", request.SchemaID),
		slog.Duration("duration", time.Since(startTime)),
	)

	// Record metrics
	s.metricsCollector.RecordLatency("validate", float64(time.Since(startTime).Milliseconds()))

	return &dto.ValidateResponse{
		Valid:    true,
		SchemaID: request.SchemaID,
	}, nil
}

// List retrieves all versions of a schema by name.
// Returns a slice of schema details or an error.
func (s *SchemaService) List(ctx context.Context, schemaName string) ([]*dto.SchemaResponse, error) {
	s.logger.InfoContext(
		ctx,
		"Listing schemas",
		slog.String("schemaName", schemaName),
	)

	schemas, err := s.schemaRepository.List(ctx, schemaName)
	if err != nil {
		s.logger.ErrorContext(
			ctx,
			"Failed to list schemas",
			slog.String("schemaName", schemaName),
			slog.Any("error", err),
		)
		s.metricsCollector.RecordError("List", "retrieval")
		return nil, fmt.Errorf("failed to list schemas: %w", err)
	}

	return dto.FromEntitySchemas(schemas), nil
}
