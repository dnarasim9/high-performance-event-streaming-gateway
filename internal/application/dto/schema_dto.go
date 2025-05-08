package dto

import (
	"time"

	"github.com/dheemanth-hn/event-streaming-gateway/internal/domain/entity"
)

// RegisterSchemaRequest is a DTO for registering a new schema.
type RegisterSchemaRequest struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Format     string `json:"format"` // JSONSchema, Avro, Protobuf
	Definition string `json:"definition"`
}

// ToEntity converts a RegisterSchemaRequest to a domain Schema entity.
// Returns an error if the schema fails validation.
func (r *RegisterSchemaRequest) ToEntity(id string) (*entity.Schema, error) {
	var format entity.SchemaFormat
	switch r.Format {
	case "Avro":
		format = entity.SchemaFormatAvro
	case "Protobuf":
		format = entity.SchemaFormatProtobuf
	case "JSONSchema":
		format = entity.SchemaFormatJSONSchema
	default:
		format = entity.SchemaFormatJSONSchema
	}

	return entity.NewSchema(id, r.Name, r.Version, format, r.Definition)
}

// SchemaResponse is a DTO for schema details in responses.
type SchemaResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Version    string    `json:"version"`
	Format     string    `json:"format"`
	Definition string    `json:"definition"`
	CreatedAt  time.Time `json:"createdAt"`
}

// FromEntitySchema creates a SchemaResponse from a domain Schema entity.
func FromEntitySchema(schema *entity.Schema) *SchemaResponse {
	return &SchemaResponse{
		ID:         schema.ID(),
		Name:       schema.Name(),
		Version:    schema.Version(),
		Format:     schema.Format().String(),
		Definition: schema.Definition(),
		CreatedAt:  schema.CreatedAt(),
	}
}

// FromEntitySchemas creates multiple SchemaResponse DTOs from domain Schema entities.
func FromEntitySchemas(schemas []*entity.Schema) []*SchemaResponse {
	responses := make([]*SchemaResponse, len(schemas))
	for i, schema := range schemas {
		responses[i] = FromEntitySchema(schema)
	}
	return responses
}

// ValidateRequest is a DTO for validating event data against a schema.
type ValidateRequest struct {
	SchemaID string `json:"schemaID"`
	Data     []byte `json:"data"`
}

// ValidateResponse is a DTO for the response to a validation request.
type ValidateResponse struct {
	Valid    bool   `json:"valid"`
	Error    string `json:"error,omitempty"`
	SchemaID string `json:"schemaID"`
}
