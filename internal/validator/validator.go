// Package validator provides contract validation against OpenAPI specs.
package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/EdgarOrtegaRamirez/apicontract/internal/parser"
)

// Result represents a single validation result.
type Result struct {
	Endpoint string
	Method   string
	Path     string
	Status   int
	Valid    bool
	Issues   []Issue
	Duration time.Duration
}

// Issue represents a contract violation.
type Issue struct {
	Severity Severity
	Category Category
	Message  string
	Detail   string
}

// Severity levels.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Category for issues.
type Category string

const (
	CategoryStatus  Category = "status"
	CategoryHeaders Category = "headers"
	CategorySchema  Category = "schema"
	CategoryContent Category = "content"
)

// Validator validates HTTP responses against an OpenAPI spec.
type Validator struct {
	Spec *parser.Spec
}

// NewValidator creates a new validator.
func NewValidator(spec *parser.Spec) *Validator {
	return &Validator{Spec: spec}
}

// ValidateEndpoint sends a request to an endpoint and validates the response.
func (v *Validator) ValidateEndpoint(endpoint parser.Endpoint, baseURL string, opts ...ValidateOption) (*Result, error) {
	start := time.Now()
	options := applyOptions(opts)

	// Build the full URL by replacing path parameters.
	fullPath := v.buildURL(endpoint.Path, options.PathParams)

	// Make the request.
	url := baseURL + fullPath
	req, err := http.NewRequest(strings.ToUpper(endpoint.Method), url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers from options.
	for k, v := range options.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: options.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	duration := time.Since(start)

	// Validate the response.
	result := &Result{
		Endpoint: endpoint.String(),
		Method:   endpoint.Method,
		Path:     endpoint.Path,
		Status:   resp.StatusCode,
		Valid:    true,
		Duration: duration,
	}

	// Check status code.
	v.checkStatus(result, endpoint, resp.StatusCode)

	// Check headers.
	v.checkHeaders(result, endpoint, resp)

	// Check body schema if applicable.
	if resp.StatusCode >= 200 && resp.StatusCode < 400 && endpoint.Operation.Responses != nil {
		v.checkBody(result, endpoint, resp)
	}

	return result, nil
}

// ValidateSpec checks the spec for structural issues.
func (v *Validator) ValidateSpec() []Issue {
	var issues []Issue

	// Check for required fields.
	if v.Spec.Info.Title == "" {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  "missing required field: info.title",
		})
	}
	if v.Spec.Info.Version == "" {
		issues = append(issues, Issue{
			Severity: SeverityError,
			Category: CategorySchema,
			Message:  "missing required field: info.version",
		})
	}

	// Check paths.
	for path, pathItem := range v.Spec.Paths {
		if len(pathItem.Operations) == 0 {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Category: CategorySchema,
				Message:  fmt.Sprintf("path %s has no operations", path),
			})
		}

		// Check for required parameters in path.
		for param := range v.extractPathParams(path) {
			found := false
			for _, p := range pathItem.Parameters {
				if p.Name == param && p.In == "path" {
					found = true
					break
				}
			}
			for _, op := range pathItem.Operations {
				for _, p := range op.Parameters {
					if p.Name == param && p.In == "path" {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				issues = append(issues, Issue{
					Severity: SeverityError,
					Category: CategorySchema,
					Message:  fmt.Sprintf("path parameter %s in %s has no matching parameter definition", param, path),
				})
			}
		}

		// Check responses.
		for _, op := range pathItem.Operations {
			if len(op.Responses) == 0 {
				issues = append(issues, Issue{
					Severity: SeverityWarning,
					Category: CategorySchema,
					Message:  fmt.Sprintf("operation %s %s has no responses defined", op.OperationID, path),
				})
			}
		}
	}

	return issues
}

// checkStatus validates the response status code against the spec.
func (v *Validator) checkStatus(result *Result, endpoint parser.Endpoint, statusCode int) {
	if endpoint.Operation.Responses == nil {
		return
	}

	// Check for exact match.
	if resp, ok := endpoint.Operation.Responses[fmt.Sprintf("%d", statusCode)]; ok {
		if resp.Description == "" {
			result.Issues = append(result.Issues, Issue{
				Severity: SeverityWarning,
				Category: CategoryStatus,
				Message:  fmt.Sprintf("status %d has no description", statusCode),
			})
		}
		return
	}

	// Check for wildcard match (e.g., "2XX", "4XX").
	wildcard := fmt.Sprintf("%dXX", statusCode/100)
	if _, ok := endpoint.Operation.Responses[wildcard]; ok {
		return
	}

	// Check for "default" response.
	if _, ok := endpoint.Operation.Responses["default"]; ok {
		return
	}

	// No matching response defined — this is a contract violation.
	result.Valid = false
	result.Issues = append(result.Issues, Issue{
		Severity: SeverityError,
		Category: CategoryStatus,
		Message:  fmt.Sprintf("status %d not defined in contract", statusCode),
	})
}

// checkHeaders validates response headers against the spec.
func (v *Validator) checkHeaders(result *Result, endpoint parser.Endpoint, resp *http.Response) {
	if endpoint.Operation.Responses == nil {
		return
	}

	statusKey := fmt.Sprintf("%d", resp.StatusCode)
	respDef, ok := endpoint.Operation.Responses[statusKey]
	if !ok {
		// Try wildcard.
		wildcard := fmt.Sprintf("%dXX", resp.StatusCode/100)
		respDef, ok = endpoint.Operation.Responses[wildcard]
		if !ok {
			return
		}
	}

	for name, headerDef := range respDef.Headers {
		actual := resp.Header.Get(name)
		if actual == "" {
			// Header is optional unless marked required (OpenAPI 3 doesn't have this).
			continue
		}

		// Validate header schema if defined.
		if headerDef.Schema != nil {
			v.validateHeaderValue(result, name, headerDef.Schema, actual)
		}
	}
}

// validateHeaderValue checks a header value against its schema.
func (v *Validator) validateHeaderValue(result *Result, name string, schema *parser.Schema, value string) {
	if schema.Type == "string" && schema.Pattern != "" {
		matched, err := regexp.MatchString(schema.Pattern, value)
		if err != nil || !matched {
			result.Valid = false
			result.Issues = append(result.Issues, Issue{
				Severity: SeverityError,
				Category: CategoryHeaders,
				Message:  fmt.Sprintf("header %s does not match pattern %s", name, schema.Pattern),
			})
		}
	}
}

// checkBody validates the response body against the response schema.
func (v *Validator) checkBody(result *Result, endpoint parser.Endpoint, resp *http.Response) {
	if endpoint.Operation.Responses == nil {
		return
	}

	statusKey := fmt.Sprintf("%d", resp.StatusCode)
	respDef, ok := endpoint.Operation.Responses[statusKey]
	if !ok {
		wildcard := fmt.Sprintf("%dXX", resp.StatusCode/100)
		respDef, ok = endpoint.Operation.Responses[wildcard]
		if !ok {
			return
		}
	}

	if respDef.Content == nil {
		return
	}

	// Get the content type.
	contentType := resp.Header.Get("Content-Type")
	if ct := respDef.Content[contentType]; ct.Schema != nil {
		v.validateBodySchema(result, ct.Schema, resp)
		return
	}

	// Try application/json as fallback.
	if ct := respDef.Content["application/json"]; ct.Schema != nil {
		v.validateBodySchema(result, ct.Schema, resp)
		return
	}
}

// validateBodySchema validates a JSON body against a schema.
func (v *Validator) validateBodySchema(result *Result, schema *parser.Schema, resp *http.Response) {
	// Parse JSON body.
	var data any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		result.Valid = false
		result.Issues = append(result.Issues, Issue{
			Severity: SeverityError,
			Category: CategoryContent,
			Message:  fmt.Sprintf("invalid JSON body: %v", err),
		})
		return
	}

	// Validate against schema.
	vs := &SchemaValidator{Schema: schema}
	if issues := vs.Validate(data); len(issues) > 0 {
		result.Valid = false
		result.Issues = append(result.Issues, issues...)
	}
}

// buildURL replaces path parameters with values from options.
func (v *Validator) buildURL(path string, params map[string]string) string {
	result := path
	for key, value := range params {
		result = strings.ReplaceAll(result, "{"+key+"}", value)
		result = strings.ReplaceAll(result, ":"+key, value)
	}
	return result
}

// extractPathParams extracts path parameter names from a path string.
func (v *Validator) extractPathParams(path string) map[string]bool {
	params := make(map[string]bool)
	re := regexp.MustCompile(`\{(\w+)\}|:(\w+)`)
	matches := re.FindAllStringSubmatch(path, -1)
	for _, m := range matches {
		if m[1] != "" {
			params[m[1]] = true
		} else if m[2] != "" {
			params[m[2]] = true
		}
	}
	return params
}
