// Package differ provides API diffing between two OpenAPI specifications.
package differ

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EdgarOrtegaRamirez/apicontract/internal/parser"
)

// ChangeType represents the type of change.
type ChangeType string

const (
	ChangeAdded    ChangeType = "added"
	ChangeRemoved  ChangeType = "removed"
	ChangeModified ChangeType = "modified"
	ChangeBreaking ChangeType = "breaking"
)

// Change represents a single API change.
type Change struct {
	Type       ChangeType
	Scope      string // endpoint, parameter, response, schema
	Severity   string // breaking, non-breaking, informational
	Path       string
	Method     string
	Detail     string
}

// DiffResult holds the diff between two specs.
type DiffResult struct {
	Changes      []Change
	Breaking     int
	NonBreaking  int
	Added        int
	Removed      int
	Modified     int
	Summary      string
}

// Diff compares two OpenAPI specifications and returns the changes.
func Diff(oldSpec, newSpec *parser.Spec) *DiffResult {
	result := &DiffResult{}

	// Collect all endpoints from both specs.
	oldEndpoints := collectEndpoints(oldSpec)
	newEndpoints := collectEndpoints(newSpec)

	// Find added, removed, and modified endpoints.
	for key, newEP := range newEndpoints {
		oldEP, exists := oldEndpoints[key]
		if !exists {
			result.Changes = append(result.Changes, Change{
				Type:     ChangeAdded,
				Scope:    "endpoint",
				Severity: "non-breaking",
				Path:     newEP.Path,
				Method:   newEP.Method,
				Detail:   fmt.Sprintf("New endpoint: %s %s", newEP.Method, newEP.Path),
			})
			result.Added++
			continue
		}

		// Compare endpoints.
		if changes := compareEndpoints(oldEP, newEP); len(changes) > 0 {
			for _, c := range changes {
				result.Changes = append(result.Changes, c)
			}
			result.Modified++
		}
	}

	for key, oldEP := range oldEndpoints {
		if _, exists := newEndpoints[key]; !exists {
			result.Changes = append(result.Changes, Change{
				Type:     ChangeRemoved,
				Scope:    "endpoint",
				Severity: "breaking",
				Path:     oldEP.Path,
				Method:   oldEP.Method,
				Detail:   fmt.Sprintf("Removed endpoint: %s %s", oldEP.Method, oldEP.Path),
			})
			result.Removed++
		}
	}

	// Sort changes: breaking first, then by type.
	sort.Slice(result.Changes, func(i, j int) bool {
		severityOrder := map[string]int{"breaking": 0, "non-breaking": 1, "informational": 2}
		si := severityOrder[result.Changes[i].Severity]
		sj := severityOrder[result.Changes[j].Severity]
		if si != sj {
			return si < sj
		}
		return result.Changes[i].Path < result.Changes[j].Path
	})

	// Calculate totals.
	for _, c := range result.Changes {
		if c.Severity == "breaking" {
			result.Breaking++
		} else if c.Severity == "non-breaking" {
			result.NonBreaking++
		}
	}

	result.Summary = generateSummary(result)
	return result
}

// EndpointKey creates a unique key for an endpoint.
func EndpointKey(method, path string) string {
	return strings.ToLower(method) + "|" + path
}

// collectEndpoints collects all endpoints from a spec.
func collectEndpoints(spec *parser.Spec) map[string]parser.Endpoint {
	endpoints := make(map[string]parser.Endpoint)
	for path, pathItem := range spec.Paths {
		for method, op := range pathItem.Operations {
			key := EndpointKey(method, path)
			endpoints[key] = parser.Endpoint{
				Method:    method,
				Path:      path,
				Operation: op,
			}
		}
	}
	return endpoints
}

// compareEndpoints compares two endpoints and returns changes.
func compareEndpoints(oldEP, newEP parser.Endpoint) []Change {
	var changes []Change

	// Compare parameters.
	oldParams := paramMap(oldEP.Operation.Parameters)
	newParams := paramMap(newEP.Operation.Parameters)

	for name, oldP := range oldParams {
		if newP, exists := newParams[name]; exists {
			if oldP.In != newP.In {
				changes = append(changes, Change{
					Type:     ChangeModified,
					Scope:    "parameter",
					Severity: "breaking",
					Path:     oldEP.Path,
					Method:   oldEP.Method,
					Detail:   fmt.Sprintf("Parameter %s 'in' changed: %s -> %s", name, oldP.In, newP.In),
				})
			}
			if !oldP.Required && newP.Required {
				changes = append(changes, Change{
					Type:     ChangeModified,
					Scope:    "parameter",
					Severity: "breaking",
					Path:     oldEP.Path,
					Method:   oldEP.Method,
					Detail:   fmt.Sprintf("Parameter %s became required", name),
				})
			}
			if oldP.Schema != nil && newP.Schema != nil {
				if schemaDiff := compareSchemas(oldP.Schema, newP.Schema, fmt.Sprintf("param:%s", name)); len(schemaDiff) > 0 {
					changes = append(changes, schemaDiff...)
				}
			}
		} else {
			changes = append(changes, Change{
				Type:     ChangeRemoved,
				Scope:    "parameter",
				Severity: "breaking",
				Path:     oldEP.Path,
				Method:   oldEP.Method,
				Detail:   fmt.Sprintf("Removed parameter: %s", name),
			})
		}
	}

	for name, newP := range newParams {
		if _, exists := oldParams[name]; !exists {
			_ = newP
			changes = append(changes, Change{
				Type:     ChangeAdded,
				Scope:    "parameter",
				Severity: "non-breaking",
				Path:     newEP.Path,
				Method:   newEP.Method,
				Detail:   fmt.Sprintf("Added parameter: %s", name),
			})
		}
	}

	// Compare responses.
	oldResp := responseMap(oldEP.Operation.Responses)
	newResp := responseMap(newEP.Operation.Responses)

	for code, newR := range newResp {
		oldR, exists := oldResp[code]
		if !exists {
			changes = append(changes, Change{
				Type:     ChangeAdded,
				Scope:    "response",
				Severity: "non-breaking",
				Path:     newEP.Path,
				Method:   newEP.Method,
				Detail:   fmt.Sprintf("Added response: %s", code),
			})
			continue
		}

		// Compare response schemas.
		if oldR.Content != nil && newR.Content != nil {
			if oldCt, ok := oldR.Content["application/json"]; ok {
				if newCt, ok := newR.Content["application/json"]; ok {
					if schemaDiff := compareSchemas(oldCt.Schema, newCt.Schema, "response"); len(schemaDiff) > 0 {
						changes = append(changes, schemaDiff...)
					}
				}
			}
		}
	}

	for code, oldR := range oldResp {
		if _, exists := newResp[code]; !exists {
			_ = oldR
			changes = append(changes, Change{
				Type:     ChangeRemoved,
				Scope:    "response",
				Severity: "breaking",
				Path:     newEP.Path,
				Method:   newEP.Method,
				Detail:   fmt.Sprintf("Removed response: %s", code),
			})
		}
	}

	// Check for deprecated status change.
	if oldEP.Operation.Deprecated != nil && *oldEP.Operation.Deprecated &&
		(newEP.Operation.Deprecated == nil || !*newEP.Operation.Deprecated) {
		changes = append(changes, Change{
			Type:     ChangeModified,
			Scope:    "endpoint",
			Severity: "non-breaking",
			Path:     newEP.Path,
			Method:   newEP.Method,
			Detail:   "Endpoint un-deprecated",
		})
	}

	return changes
}

// compareSchemas compares two schemas and returns changes.
func compareSchemas(old, new *parser.Schema, path string) []Change {
	var changes []Change

	if old.Type != new.Type {
		changes = append(changes, Change{
			Type:     ChangeModified,
			Scope:    "schema",
			Severity: "breaking",
			Path:     path,
			Detail:   fmt.Sprintf("Schema type changed: %s -> %s", old.Type, new.Type),
		})
	}

	// Check required fields.
	oldReq := make(map[string]bool)
	for _, f := range old.Required {
		oldReq[f] = true
	}

	for _, f := range new.Required {
		if !oldReq[f] {
			changes = append(changes, Change{
				Type:     ChangeModified,
				Scope:    "schema",
				Severity: "breaking",
				Path:     path + "." + f,
				Detail:   fmt.Sprintf("New required field: %s", f),
			})
		}
	}

	// Check enum restrictions.
	if len(new.Enum) > 0 && len(old.Enum) > 0 {
		oldEnums := make(map[string]bool)
		for _, e := range old.Enum {
			oldEnums[fmt.Sprintf("%v", e)] = true
		}
		for _, e := range new.Enum {
			if !oldEnums[fmt.Sprintf("%v", e)] {
				changes = append(changes, Change{
					Type:     ChangeAdded,
					Scope:    "schema",
					Severity: "non-breaking",
					Path:     path + ".enum",
					Detail:   fmt.Sprintf("New enum value: %v", e),
				})
			}
		}
		// Removed enum values are breaking.
		for _, e := range old.Enum {
			found := false
			for _, e2 := range new.Enum {
				if fmt.Sprintf("%v", e) == fmt.Sprintf("%v", e2) {
					found = true
					break
				}
			}
			if !found {
				changes = append(changes, Change{
					Type:     ChangeRemoved,
					Scope:    "schema",
					Severity: "breaking",
					Path:     path + ".enum",
					Detail:   fmt.Sprintf("Removed enum value: %v", e),
				})
			}
		}
	}

	return changes
}

// paramMap converts a slice of parameters to a map.
func paramMap(params []parser.Parameter) map[string]parser.Parameter {
	m := make(map[string]parser.Parameter)
	for _, p := range params {
		m[p.Name] = p
	}
	return m
}

// responseMap converts a map of responses.
func responseMap(responses map[string]parser.Response) map[string]parser.Response {
	if responses == nil {
		return make(map[string]parser.Response)
	}
	return responses
}

// generateSummary creates a human-readable summary.
func generateSummary(result *DiffResult) string {
	parts := []string{}
	if result.Added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", result.Added))
	}
	if result.Removed > 0 {
		parts = append(parts, fmt.Sprintf("%d removed", result.Removed))
	}
	if result.Modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", result.Modified))
	}
	if len(parts) == 0 {
		return "No changes detected"
	}
	return strings.Join(parts, ", ")
}
