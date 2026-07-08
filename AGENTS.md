# AGENTS.md

## Project Overview
**APIContract** is a CLI toolkit for validating, diffing, and generating code from OpenAPI/Swagger specifications. Written in Go with no runtime dependencies beyond standard library + cobra + yaml.v3.

## Tech Stack
- **Language:** Go 1.25
- **CLI Framework:** cobra 1.10
- **Serialization:** yaml.v3
- **Tests:** `go test ./...`

## Project Structure
```
apicontract/
├── cmd/apicontract/main.go   # CLI entry point
├── internal/
│   ├── parser/               # OpenAPI spec parser (YAML + JSON)
│   ├── validator/            # Contract + schema validation
│   ├── differ/               # API diff engine
│   ├── generator/            # Client code generator
│   └── cli/                  # CLI command definitions
├── go.mod
├── go.sum
├── README.md
├── LICENSE
└── AGENTS.md
```

## Build & Test
```bash
go build -o apicontract ./cmd/apicontract/
go test ./... -v
go vet ./...
```

## Key Design Decisions
1. **No external HTTP client** — Uses `net/http` standard library for runtime validation
2. **JSON Schema subset** — Validates only the JSON Schema subset used in OpenAPI (no $ref, no allOf/anyOf/oneOf)
3. **Deterministic output** — All map iteration uses sorted keys for reproducible CLI output
4. **YAML + JSON support** — Parses both formats, stores as structured data, no raw JSON needed for CLI

## Common Pitfalls
- **Paths must use raw JSON for operations** — The parser uses a two-pass approach: YAML unmarshal for metadata, then raw JSON for paths to preserve HTTP method keys
- **Go template generation uses fmt.Sprintf, not f-strings** — Go doesn't have Python-style f-strings
