package valueobject

import (
	"fmt"
)

// TransformationType represents the type of transformation to apply to events.
type TransformationType int

// TransformationType constants define the supported transformation types.
const (
	TransformationTypeNone TransformationType = iota
	TransformationTypeJQ
	TransformationTypeCEL
	TransformationTypeTemplate
)

// String returns the string representation of TransformationType.
func (t TransformationType) String() string {
	switch t {
	case TransformationTypeJQ:
		return "JQ"
	case TransformationTypeCEL:
		return "CEL"
	case TransformationTypeTemplate:
		return "Template"
	case TransformationTypeNone:
		return "None"
	default:
		return "Unknown"
	}
}

// Transformation is a value object that represents a transformation to apply to events.
// Transformations can be used to enrich, filter, or modify event data.
type Transformation struct {
	Type       TransformationType
	Expression string
}

// NewTransformation creates a new Transformation with the given type and expression.
// Returns an error if the transformation is invalid.
func NewTransformation(transformationType TransformationType, expression string) (Transformation, error) {
	t := Transformation{
		Type:       transformationType,
		Expression: expression,
	}

	if err := t.Validate(); err != nil {
		return Transformation{}, err
	}

	return t, nil
}

// Validate checks if the transformation is valid according to business rules.
// Returns an error if validation fails.
func (t Transformation) Validate() error {
	// None transformation type requires empty expression
	if t.Type == TransformationTypeNone {
		if t.Expression != "" {
			return fmt.Errorf("transformation type None must have empty expression")
		}
		return nil
	}

	// All other types require a non-empty expression
	if t.Expression == "" {
		return fmt.Errorf("transformation expression cannot be empty for type %s", t.Type)
	}

	// TODO: Add type-specific validation
	// - For JQ: validate JQ syntax
	// - For CEL: validate CEL syntax
	// - For Template: validate template syntax

	return nil
}

// IsNone returns true if this is a no-op transformation.
func (t Transformation) IsNone() bool {
	return t.Type == TransformationTypeNone
}

// String returns a string representation of the transformation.
func (t Transformation) String() string {
	if t.Expression == "" {
		return fmt.Sprintf("Transformation{Type: %s}", t.Type)
	}
	return fmt.Sprintf("Transformation{Type: %s, Expression: %s}", t.Type, t.Expression)
}
