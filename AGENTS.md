# AGENTS

This repository is initialized for PolarDoc development.

## Scope

- Language: Go
- Domains: PDF and OFD (kept separate)
- Entry points: CLI and MCP server

## Minimal Structure

- `cmd/polardoc`
- `cmd/polardoc-mcp`
- `internal/pdf`
- `internal/ofd`
- `internal/doc`
- `internal/app`
- `internal/render`
- `internal/security`
- `internal/mcp`
- `docs/`
- `testdata/pdf`
- `testdata/ofd`

## Rule

Do not flatten PDF and OFD semantics into a single document model.
