package differ

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/apicontract/internal/parser"
)

func TestDiff_AddedEndpoint(t *testing.T) {
	oldSpec := &parser.Spec{
		Info: parser.Info{Title: "Old", Version: "1.0.0"},
		Paths: map[string]parser.Path{
			"/users": {
				Operations: map[string]parser.Operation{
					"get": {OperationID: "listUsers"},
				},
			},
		},
	}
	newSpec := &parser.Spec{
		Info: parser.Info{Title: "New", Version: "2.0.0"},
		Paths: map[string]parser.Path{
			"/users": {
				Operations: map[string]parser.Operation{
					"get": {OperationID: "listUsers"},
				},
			},
			"/posts": {
				Operations: map[string]parser.Operation{
					"get": {OperationID: "listPosts"},
				},
			},
		},
	}

	result := Diff(oldSpec, newSpec)
	if result.Added != 1 {
		t.Errorf("expected 1 added, got %d", result.Added)
	}
	if result.Summary == "No changes detected" {
		t.Error("expected summary with changes")
	}
}

func TestDiff_RemovedEndpoint(t *testing.T) {
	oldSpec := &parser.Spec{
		Info: parser.Info{Title: "Old", Version: "1.0.0"},
		Paths: map[string]parser.Path{
			"/users": {
				Operations: map[string]parser.Operation{
					"get": {OperationID: "listUsers"},
				},
			},
			"/posts": {
				Operations: map[string]parser.Operation{
					"get": {OperationID: "listPosts"},
				},
			},
		},
	}
	newSpec := &parser.Spec{
		Info: parser.Info{Title: "New", Version: "2.0.0"},
		Paths: map[string]parser.Path{
			"/users": {
				Operations: map[string]parser.Operation{
					"get": {OperationID: "listUsers"},
				},
			},
		},
	}

	result := Diff(oldSpec, newSpec)
	if result.Removed != 1 {
		t.Errorf("expected 1 removed, got %d", result.Removed)
	}
}

func TestDiff_NoChanges(t *testing.T) {
	spec := &parser.Spec{
		Info: parser.Info{Title: "Same", Version: "1.0.0"},
		Paths: map[string]parser.Path{
			"/users": {
				Operations: map[string]parser.Operation{
					"get": {OperationID: "listUsers"},
				},
			},
		},
	}

	result := Diff(spec, spec)
	if result.Added != 0 || result.Removed != 0 || result.Modified != 0 {
		t.Errorf("expected no changes, got added=%d removed=%d modified=%d", result.Added, result.Removed, result.Modified)
	}
}

func TestEndpointKey(t *testing.T) {
	key := EndpointKey("get", "/users/{id}")
	if key != "get|/users/{id}" {
		t.Errorf("expected 'get|/users/{id}', got '%s'", key)
	}
}

func TestGenerateSummary(t *testing.T) {
	result := &DiffResult{Added: 3, Removed: 1, Modified: 2}
	summary := generateSummary(result)
	if summary == "No changes detected" {
		t.Error("expected summary with changes")
	}
}
