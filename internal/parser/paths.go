package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Known HTTP methods.
var httpMethods = map[string]bool{
	"get": true, "post": true, "put": true, "patch": true,
	"delete": true, "head": true, "options": true,
}

// ParsePaths parses the paths section from raw spec data.
func (s *Spec) ParsePaths() error {
	if s.Raw == nil {
		return fmt.Errorf("no raw data to parse paths from")
	}

	pathsRaw, ok := s.Raw["paths"]
	if !ok {
		return nil
	}

	pathsBytes, err := json.Marshal(pathsRaw)
	if err != nil {
		return fmt.Errorf("failed to marshal paths: %w", err)
	}

	var pathsMap map[string]json.RawMessage
	if err := json.Unmarshal(pathsBytes, &pathsMap); err != nil {
		return fmt.Errorf("failed to unmarshal paths: %w", err)
	}

	s.Paths = make(map[string]Path)

	for path, rawPath := range pathsMap {
		pathItem, err := parsePathItem(rawPath)
		if err != nil {
			return fmt.Errorf("failed to parse path %s: %w", path, err)
		}
		s.Paths[path] = pathItem
	}

	return nil
}

// parsePathItem parses a single path item from raw JSON.
func parsePathItem(raw json.RawMessage) (Path, error) {
	var p Path
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, fmt.Errorf("failed to unmarshal path item: %w", err)
	}

	// Parse operations from the path item.
	// Operations are keys like "get", "post", "put", etc.
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return p, fmt.Errorf("failed to parse path item as map: %w", err)
	}

	p.Operations = make(map[string]Operation)

	for key, rawOp := range rawMap {
		lower := strings.ToLower(key)
		if !httpMethods[lower] {
			continue
		}

		var op Operation
		if err := json.Unmarshal(rawOp, &op); err != nil {
			return p, fmt.Errorf("failed to unmarshal %s operation: %w", lower, err)
		}
		p.Operations[lower] = op
	}

	return p, nil
}
