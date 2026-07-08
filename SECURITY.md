# Security

## Input Validation
- Spec files are validated for structural correctness before processing
- File paths are validated to prevent path traversal
- HTTP requests use a 30-second timeout by default

## Threat Model
- **Spec file injection**: The parser uses safe YAML/JSON unmarshaling with no arbitrary code execution
- **HTTP request injection**: URLs are validated and requests use standard `net/http` with timeouts
- **Path traversal**: Output file paths are sanitized

## Dependencies
| Dependency | Purpose | Version |
|-----------|---------|---------|
| cobra | CLI framework | v1.10.2 |
| yaml.v3 | YAML parsing | v3.0.1 |
