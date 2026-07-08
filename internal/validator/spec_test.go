package validator

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/apicontract/internal/parser"
)

func TestValidateBasicSpec(t *testing.T) {
	// Missing title.
	spec := &parser.Spec{
		Info: parser.Info{Version: "1.0.0"},
		Paths: map[string]parser.Path{
			"/test": {},
		},
	}
	v := NewValidator(spec)
	issues := v.ValidateSpec()
	if len(issues) == 0 {
		t.Error("expected issues for missing title")
	}

	// Missing version.
	spec.Info = parser.Info{Title: "Test"}
	issues = v.ValidateSpec()
	found := false
	for _, i := range issues {
		if i.Message == "missing required field: info.version" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected issue for missing version")
	}
}

func TestValidatePathParams(t *testing.T) {
	spec := &parser.Spec{
		Info: parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]parser.Path{
			"/users/{id}": {},
		},
	}
	v := NewValidator(spec)
	issues := v.ValidateSpec()
	found := false
	for _, i := range issues {
		if i.Message == "path parameter id in /users/{id} has no matching parameter definition" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected issue for missing path param definition")
	}
}
