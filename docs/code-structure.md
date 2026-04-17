# Code Structure

## Package Responsibilities

- `internal/doc`: shared capability contracts and lightweight cross-format types
- `internal/pdf`: PDF implementations of doc contracts
- `internal/ofd`: OFD implementations of doc contracts
- `internal/app`: application-level routing and service wiring
- `internal/mcp`: MCP-facing interfaces and adapters
- `internal/render`: preview rendering interfaces
- `internal/security`: signing and security interfaces

## Dependency Direction

The dependency path is one-way:

- interface entry points (`cmd/polardoc`, `cmd/polardoc-mcp`, `internal/mcp`)
- `internal/app`
- `internal/doc` contracts
- format implementations (`internal/pdf`, `internal/ofd`)

`internal/pdf` and `internal/ofd` must not depend on each other.

## Why Interfaces Stay Small

Interfaces are behavior-oriented and narrow to keep boundaries explicit.
Small contracts make capabilities composable and reduce coupling across packages.

## Why There Is No Giant Document Model

PolarDoc does not define one unified in-memory model for all formats.
PDF and OFD keep independent semantics and data structures.
Shared contracts unify operations, not internal format representation.
