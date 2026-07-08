package validator

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/apicontract/internal/parser"
)

func TestValidateResponse(t *testing.T) {
	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"id":     {Type: "integer"},
			"name":   {Type: "string", MinLength: PtrInt(1)},
		},
		Required: []string{"id", "name"},
	}

	// Valid response.
	body := []byte(`{"id": 1, "name": "test"}`)
	issues := ValidateResponse(body, schema)
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}

	// Missing required field.
	body = []byte(`{"id": 1}`)
	issues = ValidateResponse(body, schema)
	if len(issues) == 0 {
		t.Error("expected issue for missing 'name' field")
	}

	// Invalid type.
	body = []byte(`{"id": "not-int", "name": "test"}`)
	issues = ValidateResponse(body, schema)
	if len(issues) == 0 {
		t.Error("expected issue for invalid type")
	}

	// Invalid JSON.
	body = []byte(`{invalid}`)
	issues = ValidateResponse(body, schema)
	if len(issues) == 0 {
		t.Error("expected issue for invalid JSON")
	}
}

func TestValidateStringValue(t *testing.T) {
	schema := &parser.Schema{
		Type:      "string",
		MinLength: PtrInt(3),
		MaxLength: PtrInt(10),
	}

	tests := []struct {
		name     string
		value    any
		expected int
	}{
		{"valid", "abc", 0},
		{"too short", "ab", 1},
		{"too long", "abcdefghijk", 1},
		{"wrong type", 123, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vs := &SchemaValidator{Schema: schema}
			issues := vs.Validate(tt.value)
			if len(issues) != tt.expected {
				t.Errorf("expected %d issues, got %d: %v", tt.expected, len(issues), issues)
			}
		})
	}
}

func TestValidateNumberValue(t *testing.T) {
	schema := &parser.Schema{
		Type:    "number",
		Minimum: PtrFloat(0),
		Maximum: PtrFloat(100),
	}

	vs := &SchemaValidator{Schema: schema}

	tests := []struct {
		name     string
		value    any
		expected int
	}{
		{"valid", float64(50), 0},
		{"too low", float64(-1), 1},
		{"too high", float64(101), 1},
		{"wrong type", "string", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := vs.Validate(tt.value)
			if len(issues) != tt.expected {
				t.Errorf("expected %d issues, got %d: %v", tt.expected, len(issues), issues)
			}
		})
	}
}

func TestValidateObjectWithNestedSchema(t *testing.T) {
	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"address": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"city": {Type: "string"},
				},
				Required: []string{"city"},
			},
		},
		Required: []string{"address"},
	}

	body := []byte(`{"address": {"street": "123 Main St"}}`)
	issues := ValidateResponse(body, schema)
	if len(issues) == 0 {
		t.Error("expected issue for missing 'city' field in nested object")
	}
}

func TestValidateArray(t *testing.T) {
	schema := &parser.Schema{
		Type: "array",
		Items: &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"name": {Type: "string"},
			},
			Required: []string{"name"},
		},
	}

	tests := []struct {
		name     string
		value    any
		expected int
	}{
		{"valid array", []any{map[string]any{"name": "a"}, map[string]any{"name": "b"}}, 0},
		{"invalid items", []any{map[string]any{"id": 1}}, 1},
	}

	vs := &SchemaValidator{Schema: schema}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := vs.Validate(tt.value)
			if len(issues) != tt.expected {
				t.Errorf("expected %d issues, got %d: %v", tt.expected, len(issues), issues)
			}
		})
	}
}

func TestValidateResponseField(t *testing.T) {
	schema := &parser.Schema{Type: "integer"}
	body := []byte(`{"user": {"id": 42, "name": "test"}}`)

	issues := ValidateResponseField(body, "user.id", schema)
	if len(issues) != 0 {
		t.Errorf("expected no issues for valid field, got %v", issues)
	}

	issues = ValidateResponseField(body, "user.missing", schema)
	if len(issues) == 0 {
		t.Error("expected issue for missing field")
	}

	issues = ValidateResponseField(body, "user.id", &parser.Schema{Type: "string"})
	if len(issues) == 0 {
		t.Error("expected issue for type mismatch")
	}
}

func TestTypeOf(t *testing.T) {
	tests := []struct {
		input  any
		expect string
	}{
		{nil, "null"},
		{"string", "string"},
		{float64(1), "number"},
		{true, "boolean"},
		{map[string]any{}, "object"},
		{[]any{}, "array"},
	}

	for _, tt := range tests {
		t.Run(tt.expect, func(t *testing.T) {
			got := typeOf(tt.input)
			if got != tt.expect {
				t.Errorf("typeOf(%v) = %s, want %s", tt.input, got, tt.expect)
			}
		})
	}
}

func PtrInt(i int) *int { return &i }
func PtrFloat(f float64) *float64 { return &f }
