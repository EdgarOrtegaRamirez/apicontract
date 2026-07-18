package parser

import (
	"strings"
	"testing"
)

func TestParseSpecFromBytes(t *testing.T) {
	specYAML := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: Success
`
	spec, err := ParseSpecFromBytes([]byte(specYAML))
	if err != nil {
		t.Fatalf("failed to parse spec: %v", err)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("expected title 'Test API', got '%s'", spec.Info.Title)
	}
	if spec.Info.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", spec.Info.Version)
	}
	if len(spec.Paths) != 1 {
		t.Errorf("expected 1 path, got %d", len(spec.Paths))
	}
}

func TestParseSpecFromBytes_JSON(t *testing.T) {
	specJSON := `{
		"openapi": "3.0.0",
		"info": {"title": "JSON API", "version": "2.0.0"},
		"paths": {"/items": {"get": {"operationId": "getItems", "responses": {"200": {"description": "OK"}}}}}
	}`
	spec, err := ParseSpecFromBytes([]byte(specJSON))
	if err != nil {
		t.Fatalf("failed to parse JSON spec: %v", err)
	}
	if spec.Info.Title != "JSON API" {
		t.Errorf("expected 'JSON API', got '%s'", spec.Info.Title)
	}
}

func TestParseSpecFromBytes_Invalid(t *testing.T) {
	_, err := ParseSpecFromBytes([]byte("not: valid: yaml: json: {"))
	if err == nil {
		t.Error("expected error for invalid spec")
	}
}

func TestGetEndpoints(t *testing.T) {
	spec := &Spec{
		Info: Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]Path{
			"/users": {
				Operations: map[string]Operation{
					"get":  {OperationID: "listUsers"},
					"post": {OperationID: "createUser"},
				},
			},
		},
	}

	endpoints := spec.GetEndpoints()
	if len(endpoints) != 2 {
		t.Errorf("expected 2 endpoints, got %d", len(endpoints))
	}

	for _, ep := range endpoints {
		s := ep.String()
		if !strings.Contains(s, "/users") {
			t.Errorf("endpoint string should contain '/users': %s", s)
		}
	}
}

func TestValidateBasic(t *testing.T) {
	spec := &Spec{
		Info: Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]Path{
			"/test": {},
		},
	}
	issues := spec.ValidateBasic()
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}

	spec.Info.Title = ""
	issues = spec.ValidateBasic()
	if len(issues) != 1 {
		t.Errorf("expected 1 issue for missing title, got %d", len(issues))
	}
}

func TestEndpointString(t *testing.T) {
	ep := Endpoint{Method: "get", Path: "/users/123"}
	s := ep.String()
	if s != "GET /users/123" {
		t.Errorf("expected 'GET /users/123', got '%s'", s)
	}
}
