# Code Structure

## Package Responsibilities

- `internal/doc`: shared capability contracts and lightweight cross-format types
- `internal/pdf`: PDF implementations of doc contracts
- `internal/ofd`: OFD implementations of doc contracts
- `internal/app`: application-level resolver and service wiring only
- `internal/mcp`: MCP-facing interfaces and adapters
- `internal/render`: preview rendering interfaces
- `internal/security`: signing and security interfaces

## Dependency Direction

Dependencies are one-way:

- `cmd/polardoc` -> `cmd/polardoc/commands` -> `internal/app` -> `internal/doc`
- `cmd/polardoc-mcp` -> `internal/mcp` -> `internal/app` -> `internal/doc`
- `internal/pdf` -> `internal/doc`
- `internal/ofd` -> `internal/doc`

`internal/pdf` and `internal/ofd` must not depend on each other.
`internal/pdf` and `internal/ofd` now contain the first real `info` service implementations; wiring still belongs in `internal/app`.

## Why Interfaces Stay Small

Interfaces are behavior-oriented and narrow to keep boundaries explicit.
Small contracts make capabilities composable and reduce coupling across packages.

## Why There Is No Giant Document Model

PolarDoc does not define one unified in-memory model for all formats.
PDF and OFD keep independent semantics and data structures.
Shared contracts unify operations, not internal format representation.
