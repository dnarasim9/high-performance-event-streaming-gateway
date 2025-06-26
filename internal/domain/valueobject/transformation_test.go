package valueobject

import (
	"testing"
)

// TestNewTransformation_NoneType tests creating a None transformation
func TestNewTransformation_NoneType(t *testing.T) {
	trans, err := NewTransformation(TransformationTypeNone, "")
	if err != nil {
		t.Errorf("NewTransformation(None) error = %v, want nil", err)
	}
	if trans.Type != TransformationTypeNone {
		t.Errorf("NewTransformation(None) Type = %v, want TransformationTypeNone", trans.Type)
	}
	if trans.Expression != "" {
		t.Errorf("NewTransformation(None) Expression = %s, want empty", trans.Expression)
	}
}

// TestNewTransformation_JQType tests creating a JQ transformation
func TestNewTransformation_JQType(t *testing.T) {
	trans, err := NewTransformation(TransformationTypeJQ, ".field | select(.value > 10)")
	if err != nil {
		t.Errorf("NewTransformation(JQ) error = %v, want nil", err)
	}
	if trans.Type != TransformationTypeJQ {
		t.Errorf("NewTransformation(JQ) Type = %v, want TransformationTypeJQ", trans.Type)
	}
	if trans.Expression != ".field | select(.value > 10)" {
		t.Errorf("NewTransformation(JQ) Expression mismatch")
	}
}

// TestNewTransformation_CELType tests creating a CEL transformation
func TestNewTransformation_CELType(t *testing.T) {
	trans, err := NewTransformation(TransformationTypeCEL, "message.value > 100")
	if err != nil {
		t.Errorf("NewTransformation(CEL) error = %v, want nil", err)
	}
	if trans.Type != TransformationTypeCEL {
		t.Errorf("NewTransformation(CEL) Type = %v, want TransformationTypeCEL", trans.Type)
	}
	if trans.Expression != "message.value > 100" {
		t.Errorf("NewTransformation(CEL) Expression mismatch")
	}
}

// TestNewTransformation_TemplateType tests creating a Template transformation
func TestNewTransformation_TemplateType(t *testing.T) {
	trans, err := NewTransformation(TransformationTypeTemplate, "Hello {{.Name}}")
	if err != nil {
		t.Errorf("NewTransformation(Template) error = %v, want nil", err)
	}
	if trans.Type != TransformationTypeTemplate {
		t.Errorf("NewTransformation(Template) Type = %v, want TransformationTypeTemplate", trans.Type)
	}
	if trans.Expression != "Hello {{.Name}}" {
		t.Errorf("NewTransformation(Template) Expression mismatch")
	}
}

// TestNewTransformation_NoneTypeWithExpression tests None type with non-empty expression
func TestNewTransformation_NoneTypeWithExpression(t *testing.T) {
	_, err := NewTransformation(TransformationTypeNone, "expression")
	if err == nil {
		t.Error("NewTransformation(None) with expression expected error, got nil")
	}
}

// TestNewTransformation_JQTypeWithEmptyExpression tests JQ type with empty expression
func TestNewTransformation_JQTypeWithEmptyExpression(t *testing.T) {
	_, err := NewTransformation(TransformationTypeJQ, "")
	if err == nil {
		t.Error("NewTransformation(JQ) with empty expression expected error, got nil")
	}
}

// TestNewTransformation_CELTypeWithEmptyExpression tests CEL type with empty expression
func TestNewTransformation_CELTypeWithEmptyExpression(t *testing.T) {
	_, err := NewTransformation(TransformationTypeCEL, "")
	if err == nil {
		t.Error("NewTransformation(CEL) with empty expression expected error, got nil")
	}
}

// TestNewTransformation_TemplateTypeWithEmptyExpression tests Template type with empty expression
func TestNewTransformation_TemplateTypeWithEmptyExpression(t *testing.T) {
	_, err := NewTransformation(TransformationTypeTemplate, "")
	if err == nil {
		t.Error("NewTransformation(Template) with empty expression expected error, got nil")
	}
}

// TestTransformation_Validate_Valid tests validation of valid transformation
func TestTransformation_Validate_Valid(t *testing.T) {
	tests := []struct {
		name       string
		transType  TransformationType
		expression string
	}{
		{
			name:       "valid None transformation",
			transType:  TransformationTypeNone,
			expression: "",
		},
		{
			name:       "valid JQ transformation",
			transType:  TransformationTypeJQ,
			expression: ".field",
		},
		{
			name:       "valid CEL transformation",
			transType:  TransformationTypeCEL,
			expression: "value > 100",
		},
		{
			name:       "valid Template transformation",
			transType:  TransformationTypeTemplate,
			expression: "Hello {{.Name}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans := Transformation{
				Type:       tt.transType,
				Expression: tt.expression,
			}
			err := trans.Validate()
			if err != nil {
				t.Errorf("Validate() error = %v, want nil", err)
			}
		})
	}
}

// TestTransformation_Validate_Invalid tests validation of invalid transformation
func TestTransformation_Validate_Invalid(t *testing.T) {
	tests := []struct {
		name       string
		transType  TransformationType
		expression string
	}{
		{
			name:       "None with non-empty expression",
			transType:  TransformationTypeNone,
			expression: "some-expression",
		},
		{
			name:       "JQ with empty expression",
			transType:  TransformationTypeJQ,
			expression: "",
		},
		{
			name:       "CEL with empty expression",
			transType:  TransformationTypeCEL,
			expression: "",
		},
		{
			name:       "Template with empty expression",
			transType:  TransformationTypeTemplate,
			expression: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans := Transformation{
				Type:       tt.transType,
				Expression: tt.expression,
			}
			err := trans.Validate()
			if err == nil {
				t.Errorf("Validate() expected error, got nil")
			}
		})
	}
}

// TestTransformation_IsNone tests IsNone method
func TestTransformation_IsNone(t *testing.T) {
	tests := []struct {
		name       string
		transType  TransformationType
		expression string
		expected   bool
	}{
		{
			name:       "None type",
			transType:  TransformationTypeNone,
			expression: "",
			expected:   true,
		},
		{
			name:       "JQ type",
			transType:  TransformationTypeJQ,
			expression: ".field",
			expected:   false,
		},
		{
			name:       "CEL type",
			transType:  TransformationTypeCEL,
			expression: "value > 100",
			expected:   false,
		},
		{
			name:       "Template type",
			transType:  TransformationTypeTemplate,
			expression: "Hello {{.Name}}",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans := Transformation{
				Type:       tt.transType,
				Expression: tt.expression,
			}
			if trans.IsNone() != tt.expected {
				t.Errorf("IsNone() = %v, want %v", trans.IsNone(), tt.expected)
			}
		})
	}
}

// TestTransformationType_String tests string representation of transformation type
func TestTransformationType_String(t *testing.T) {
	tests := []struct {
		transType TransformationType
		expected  string
	}{
		{TransformationTypeNone, "None"},
		{TransformationTypeJQ, "JQ"},
		{TransformationTypeCEL, "CEL"},
		{TransformationTypeTemplate, "Template"},
		{TransformationType(999), "Unknown"},
	}

	for _, tt := range tests {
		if tt.transType.String() != tt.expected {
			t.Errorf("String() = %s, want %s", tt.transType.String(), tt.expected)
		}
	}
}

// TestTransformation_String tests string representation of transformation
func TestTransformation_String(t *testing.T) {
	tests := []struct {
		name               string
		trans              Transformation
		containsExpression bool
	}{
		{
			name: "None transformation",
			trans: Transformation{
				Type:       TransformationTypeNone,
				Expression: "",
			},
			containsExpression: false,
		},
		{
			name: "JQ transformation",
			trans: Transformation{
				Type:       TransformationTypeJQ,
				Expression: ".field",
			},
			containsExpression: true,
		},
		{
			name: "CEL transformation",
			trans: Transformation{
				Type:       TransformationTypeCEL,
				Expression: "value > 100",
			},
			containsExpression: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.trans.String()
			if str == "" {
				t.Error("String() returned empty string")
			}
			if tt.containsExpression && tt.trans.Expression != "" {
				if len(str) == 0 || str == "Transformation{Type: "+tt.trans.Type.String()+"}" {
					t.Error("String() should contain expression")
				}
			}
		})
	}
}

// TestTransformation_ComplexExpressions tests with complex real-world expressions
func TestTransformation_ComplexExpressions(t *testing.T) {
	tests := []struct {
		name       string
		transType  TransformationType
		expression string
	}{
		{
			name:       "complex JQ filter",
			transType:  TransformationTypeJQ,
			expression: ".[] | select(.status == \"active\") | {id, name, email}",
		},
		{
			name:       "complex CEL expression",
			transType:  TransformationTypeCEL,
			expression: "has(message.payload) && message.payload.value > 100 && message.timestamp < now",
		},
		{
			name:       "complex template",
			transType:  TransformationTypeTemplate,
			expression: "User {{.ID}}: {{.FirstName}} {{.LastName}} <{{.Email}}>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trans, err := NewTransformation(tt.transType, tt.expression)
			if err != nil {
				t.Errorf("NewTransformation() error = %v, want nil", err)
			}
			if trans.Expression != tt.expression {
				t.Errorf("Expression mismatch: got %s, want %s", trans.Expression, tt.expression)
			}
		})
	}
}

// TestTransformation_EquivalenceCheck tests transformations for equivalence
func TestTransformation_EquivalenceCheck(t *testing.T) {
	trans1, _ := NewTransformation(TransformationTypeJQ, ".field")
	trans2, _ := NewTransformation(TransformationTypeJQ, ".field")
	trans3, _ := NewTransformation(TransformationTypeJQ, ".other")

	// Same type and expression
	if trans1.Type != trans2.Type || trans1.Expression != trans2.Expression {
		t.Error("Equivalent transformations should have same type and expression")
	}

	// Same type but different expression
	if trans1.Type == trans3.Type && trans1.Expression == trans3.Expression {
		t.Error("Different expressions should not be equal")
	}
}
