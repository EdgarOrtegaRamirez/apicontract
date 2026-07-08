package validator

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/EdgarOrtegaRamirez/apicontract/internal/parser"
)

// SchemaValidator validates data against a JSON Schema.
type SchemaValidator struct {
	Schema *parser.Schema
}

// Validate validates data against the schema, returning issues.
func (sv *SchemaValidator) Validate(data any) []Issue {
	return sv.validateValue(data, sv.Schema, "$")
}

func (sv *SchemaValidator) validateValue(data any, schema *parser.Schema, path string) []Issue {
	var issues []Issue

	if schema == nil {
		return nil
	}

	switch schema.Type {
	case "object":
		issues = append(issues, sv.validateObject(data, schema, path)...)
	case "array":
		issues = append(issues, sv.validateArray(data, schema, path)...)
	case "string":
		issues = append(issues, sv.validateString(data, schema, path)...)
	case "number", "integer":
		issues = append(issues, sv.validateNumber(data, schema, path)...)
	case "boolean":
		issues = append(issues, sv.validateBoolean(data, schema, path)...)
	}

	return issues
}

func (sv *SchemaValidator) validateObject(data any, schema *parser.Schema, path string) []Issue {
	var issues []Issue

	obj, ok := data.(map[string]any)
	if !ok {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  fmt.Sprintf("expected object at %s, got %s", path, typeOf(data)),
		})
		return issues
	}

	// Check required fields.
	for _, req := range schema.Required {
		if _, exists := obj[req]; !exists {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Category: CategorySchema,
				Message:  fmt.Sprintf("required field %s.%s is missing", path, req),
			})
		}
	}

	// Validate properties.
	for name, propSchema := range schema.Properties {
		if val, exists := obj[name]; exists {
			issues = append(issues, sv.validateValue(val, propSchema, fmt.Sprintf("%s.%s", path, name))...)
		}
	}

	return issues
}

func (sv *SchemaValidator) validateArray(data any, schema *parser.Schema, path string) []Issue {
	var issues []Issue

	arr, ok := data.([]any)
	if !ok {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  fmt.Sprintf("expected array at %s, got %s", path, typeOf(data)),
		})
		return issues
	}

	if schema.Items != nil {
		for i, item := range arr {
			issues = append(issues, sv.validateValue(item, schema.Items, fmt.Sprintf("%s[%d]", path, i))...)
		}
	}

	return issues
}

func (sv *SchemaValidator) validateString(data any, schema *parser.Schema, path string) []Issue {
	var issues []Issue

	str, ok := data.(string)
	if !ok {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  fmt.Sprintf("expected string at %s, got %s", path, typeOf(data)),
		})
		return issues
	}

	if schema.MinLength != nil && len(str) < *schema.MinLength {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  fmt.Sprintf("string at %s is too short (min %d, got %d)", path, *schema.MinLength, len(str)),
		})
	}

	if schema.MaxLength != nil && len(str) > *schema.MaxLength {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  fmt.Sprintf("string at %s is too long (max %d, got %d)", path, *schema.MaxLength, len(str)),
		})
	}

	if schema.Pattern != "" {
		// Simple pattern check (not full regex support).
		if !strings.Contains(str, schema.Pattern) {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Category: CategorySchema,
				Message:  fmt.Sprintf("string at %s may not match pattern %s", path, schema.Pattern),
			})
		}
	}

	if len(schema.Enum) > 0 {
		found := false
		for _, e := range schema.Enum {
			if fmt.Sprintf("%v", e) == str {
				found = true
				break
			}
		}
		if !found {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Category: CategorySchema,
				Message:  fmt.Sprintf("value at %s is not in enum: %v", path, schema.Enum),
			})
		}
	}

	return issues
}

func (sv *SchemaValidator) validateNumber(data any, schema *parser.Schema, path string) []Issue {
	var issues []Issue

	var num float64
	switch v := data.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	default:
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  fmt.Sprintf("expected number at %s, got %s", path, typeOf(data)),
		})
		return issues
	}

	if schema.Minimum != nil && num < *schema.Minimum {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  fmt.Sprintf("value at %s is below minimum (%.0f)", path, *schema.Minimum),
		})
	}

	if schema.Maximum != nil && num > *schema.Maximum {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  fmt.Sprintf("value at %s is above maximum (%.0f)", path, *schema.Maximum),
		})
	}

	return issues
}

func (sv *SchemaValidator) validateBoolean(data any, schema *parser.Schema, path string) []Issue {
	var issues []Issue

	if _, ok := data.(bool); !ok {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  fmt.Sprintf("expected boolean at %s, got %s", path, typeOf(data)),
		})
	}

	return issues
}

// typeOf returns a human-readable type name.
func typeOf(v any) string {
	if v == nil {
		return "null"
	}
	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Map:
		return "object"
	case reflect.Slice:
		return "array"
	default:
		return t.String()
	}
}

// ValidateResponse is a convenience function to validate a JSON response.
func ValidateResponse(body []byte, schema *parser.Schema) []Issue {
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return []Issue{{
			Severity: SeverityError,
			Category: CategoryContent,
			Message:  fmt.Sprintf("invalid JSON: %v", err),
		}}
	}

	vs := &SchemaValidator{Schema: schema}
	return vs.Validate(data)
}

// ValidateResponseField validates a specific field in a JSON response.
func ValidateResponseField(body []byte, fieldPath string, schema *parser.Schema) []Issue {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return []Issue{{
			Severity: SeverityError,
			Category: CategoryContent,
			Message:  fmt.Sprintf("invalid JSON: %v", err),
		}}
	}

	// Navigate to the field.
	parts := strings.Split(fieldPath, ".")
	current := any(data)
	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			var ok bool
			current, ok = v[part]
			if !ok {
				return []Issue{{
					Severity: SeverityError,
					Category: CategorySchema,
					Message:  fmt.Sprintf("field %s not found", fieldPath),
				}}
			}
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil {
				return []Issue{{
					Severity: SeverityError,
					Category: CategorySchema,
					Message:  fmt.Sprintf("invalid array index: %s", part),
				}}
			}
			if idx < 0 || idx >= len(v) {
				return []Issue{{
					Severity: SeverityError,
					Category: CategorySchema,
					Message:  fmt.Sprintf("array index %d out of bounds", idx),
				}}
			}
			current = v[idx]
		default:
			return []Issue{{
				Severity: SeverityError,
				Category: CategorySchema,
				Message:  fmt.Sprintf("cannot navigate into %T", current),
			}}
		}
	}

	vs := &SchemaValidator{Schema: schema}
	return vs.Validate(current)
}
