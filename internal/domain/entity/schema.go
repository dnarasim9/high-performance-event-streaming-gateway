package entity

import (
	"fmt"
	"time"
)

// SchemaFormat represents the format/type of the schema.
type SchemaFormat int

// SchemaFormat constants define the supported schema formats.
const (
	SchemaFormatJSONSchema SchemaFormat = iota
	SchemaFormatAvro
	SchemaFormatProtobuf
)

// String returns the string representation of SchemaFormat.
func (f SchemaFormat) String() string {
	switch f {
	case SchemaFormatAvro:
		return "Avro"
	case SchemaFormatProtobuf:
		return "Protobuf"
	case SchemaFormatJSONSchema:
		return "JSONSchema"
	default:
		return "Unknown"
	}
}

// Schema is a domain entity that represents a schema used to validate event data.
// Schemas are immutable after creation.
type Schema struct {
	id         string
	name       string
	version    string
	format     SchemaFormat
	definition string
	createdAt  time.Time
}

// NewSchema creates a new Schema with the given parameters and validates it.
// Returns an error if the schema fails validation.
func NewSchema(
	id string,
	name string,
	version string,
	format SchemaFormat,
	definition string,
) (*Schema, error) {
	s := &Schema{
		id:         id,
		name:       name,
		version:    version,
		format:     format,
		definition: definition,
		createdAt:  time.Now().UTC(),
	}

	if err := s.validate(); err != nil {
		return nil, err
	}

	return s, nil
}

// validate checks if the schema is valid according to business rules.
func (s *Schema) validate() error {
	if s.id == "" {
		return fmt.Errorf("schema ID cannot be empty")
	}
	if s.name == "" {
		return fmt.Errorf("schema name cannot be empty")
	}
	if s.version == "" {
		return fmt.Errorf("schema version cannot be empty")
	}
	if s.definition == "" {
		return fmt.Errorf("schema definition cannot be empty")
	}
	return nil
}

// ID returns the schema ID.
func (s *Schema) ID() string {
	return s.id
}

// Name returns the schema name.
func (s *Schema) Name() string {
	return s.name
}

// Version returns the schema version.
func (s *Schema) Version() string {
	return s.version
}

// Format returns the schema format.
func (s *Schema) Format() SchemaFormat {
	return s.format
}

// Definition returns the schema definition.
func (s *Schema) Definition() string {
	return s.definition
}

// CreatedAt returns the creation timestamp.
func (s *Schema) CreatedAt() time.Time {
	return s.createdAt
}

// ValidateEvent validates the given event data against this schema.
// Currently returns a placeholder error indicating that validation is not implemented.
// This method should be extended to support validation for different schema formats.
func (s *Schema) ValidateEvent(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("event data cannot be empty")
	}

	// TODO: Implement schema validation based on format
	// - For JSONSchema: validate using a JSON Schema validator
	// - For Avro: validate using an Avro library
	// - For Protobuf: validate using protobuf marshaling

	return nil
}
