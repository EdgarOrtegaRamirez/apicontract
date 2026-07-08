// Package parser handles loading and parsing OpenAPI/Swagger specifications.
package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Spec represents a parsed OpenAPI/Swagger specification.
type Spec struct {
	OpenAPI   string            `yaml:"openapi" json:"openapi,omitempty"`
	Swagger   string            `yaml:"swagger" json:"swagger,omitempty"`
	Info      Info              `yaml:"info" json:"info"`
	Paths     map[string]Path   `yaml:"paths" json:"paths"`
	Servers   []Server          `yaml:"servers" json:"servers,omitempty"`
	Components *Components    `yaml:"components,omitempty" json:"components,omitempty"`
	Raw       map[string]any    `yaml:"-" json:"-"`
}

// Info holds metadata about the API.
type Info struct {
	Title          string `yaml:"title" json:"title"`
	Version        string `yaml:"version" json:"version"`
	Description    string `yaml:"description,omitempty" json:"description,omitempty"`
	Contact        Contact `yaml:"contact,omitempty" json:"contact,omitempty"`
	License        License `yaml:"license,omitempty" json:"license,omitempty"`
}

// Contact holds contact information.
type Contact struct {
	Name  string `yaml:"name,omitempty" json:"name,omitempty"`
	URL   string `yaml:"url,omitempty" json:"url,omitempty"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`
}

// License holds license information.
type License struct {
	Name string `yaml:"name" json:"name"`
	URL  string `yaml:"url,omitempty" json:"url,omitempty"`
}

// Server represents an API server.
type Server struct {
	URL         string `yaml:"url" json:"url"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Path represents an OpenAPI path item.
type Path struct {
	Summary       string       `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description   string       `yaml:"description,omitempty" json:"description,omitempty"`
	Operations    map[string]Operation `yaml:"-" json:"-"`
	OperationID   string       `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Parameters    []Parameter  `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody   *RequestBody `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	Responses     map[string]Response `yaml:"responses,omitempty" json:"responses,omitempty"`
	Deprecated    *bool      `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
}

// Operation represents an HTTP operation.
type Operation struct {
	Summary     string            `yaml:"summary,omitempty" json:"summary,omitempty"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	OperationID string            `yaml:"operationId,omitempty" json:"operationId,omitempty"`
	Tags        []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Parameters  []Parameter       `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	RequestBody *RequestBody      `yaml:"requestBody,omitempty" json:"requestBody,omitempty"`
	Responses   map[string]Response `yaml:"responses,omitempty" json:"responses,omitempty"`
	Deprecated  *bool             `yaml:"deprecated,omitempty" json:"deprecated,omitempty"`
	Security    []map[string][]string `yaml:"security,omitempty" json:"security,omitempty"`
}

// Parameter represents an OpenAPI parameter.
type Parameter struct {
	Name        string   `yaml:"name" json:"name"`
	In          string   `yaml:"in" json:"in"` // path, query, header, cookie
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Schema      *Schema  `yaml:"schema,omitempty" json:"schema,omitempty"`
	Example     any      `yaml:"example,omitempty" json:"example,omitempty"`
}

// RequestBody represents an OpenAPI request body.
type RequestBody struct {
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Content     map[string]MediaType `yaml:"content" json:"content"`
	Required    bool               `yaml:"required,omitempty" json:"required,omitempty"`
}

// MediaType represents a media type in request/response body.
type MediaType struct {
	Schema   *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
	Example  any     `yaml:"example,omitempty" json:"example,omitempty"`
}

// Response represents an OpenAPI response.
type Response struct {
	Description string            `yaml:"description" json:"description"`
	Headers     map[string]Header `yaml:"headers,omitempty" json:"headers,omitempty"`
	Content     map[string]MediaType `yaml:"content,omitempty" json:"content,omitempty"`
}

// Header represents an OpenAPI response header.
type Header struct {
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Schema      *Schema `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// Schema represents a JSON Schema (subset used in OpenAPI).
type Schema struct {
	Type         string           `yaml:"type,omitempty" json:"type,omitempty"`
	Description  string           `yaml:"description,omitempty" json:"description,omitempty"`
	Format       string           `yaml:"format,omitempty" json:"format,omitempty"`
	Properties   map[string]*Schema `yaml:"properties,omitempty" json:"properties,omitempty"`
	Required     []string         `yaml:"required,omitempty" json:"required,omitempty"`
	Items        *Schema          `yaml:"items,omitempty" json:"items,omitempty"`
	Enum         []any            `yaml:"enum,omitempty" json:"enum,omitempty"`
	MinLength    *int             `yaml:"minLength,omitempty" json:"minLength,omitempty"`
	MaxLength    *int             `yaml:"maxLength,omitempty" json:"maxLength,omitempty"`
	Minimum      *float64         `yaml:"minimum,omitempty" json:"minimum,omitempty"`
	Maximum      *float64         `yaml:"maximum,omitempty" json:"maximum,omitempty"`
	Pattern      string           `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	AdditionalProperties any     `yaml:"additionalProperties,omitempty" json:"additionalProperties,omitempty"`
}

// Components holds reusable components.
type Components struct {
	Schemas map[string]*Schema `yaml:"schemas,omitempty" json:"schemas,omitempty"`
}

// ParseSpec loads and parses an OpenAPI specification from a file.
func ParseSpec(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	return ParseSpecFromBytes(data)
}

// ParseSpecFromBytes parses an OpenAPI specification from bytes.
func ParseSpecFromBytes(data []byte) (*Spec, error) {
	spec := &Spec{}

	// Try YAML first, then JSON
	if err := yaml.Unmarshal(data, spec); err != nil {
		// Try JSON
		if err2 := json.Unmarshal(data, spec); err2 != nil {
			return nil, fmt.Errorf("failed to parse spec: yaml error: %w, json error: %w", err, err2)
		}
	}

	// Parse operations from paths
	if spec.Paths == nil {
		spec.Paths = make(map[string]Path)
	}

	// Store raw data
	spec.Raw = make(map[string]any)
	if err := yaml.Unmarshal(data, spec.Raw); err != nil {
		json.Unmarshal(data, spec.Raw)
	}

	return spec, nil
}

// GetHTTPMethods returns all HTTP methods defined in the spec.
func (s *Spec) GetHTTPMethods() map[string]map[string]Operation {
	methods := make(map[string]map[string]Operation)
	for path, pathItem := range s.Paths {
		for method, op := range pathItem.Operations {
			if methods[method] == nil {
				methods[method] = make(map[string]Operation)
			}
			methods[method][path] = op
		}
	}
	return methods
}

// GetEndpoints returns all endpoints as a flat list.
func (s *Spec) GetEndpoints() []Endpoint {
	var endpoints []Endpoint
	for path, pathItem := range s.Paths {
		for method, op := range pathItem.Operations {
			endpoints = append(endpoints, Endpoint{
				Method:    method,
				Path:      path,
				Operation: op,
			})
		}
	}
	return endpoints
}

// Endpoint represents a single API endpoint.
type Endpoint struct {
	Method    string
	Path      string
	Operation Operation
}

// String returns a human-readable representation of the endpoint.
func (e Endpoint) String() string {
	return fmt.Sprintf("%s %s", strings.ToUpper(e.Method), e.Path)
}

// ValidateBasic checks for common spec issues.
func (s *Spec) ValidateBasic() []string {
	var issues []string

	if s.Info.Title == "" {
		issues = append(issues, "missing 'info.title'")
	}
	if s.Info.Version == "" {
		issues = append(issues, "missing 'info.version'")
	}
	if len(s.Paths) == 0 {
		issues = append(issues, "no paths defined")
	}

	return issues
}
