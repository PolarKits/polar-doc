# Architecture

## Why Abstraction Is Needed

A shared abstraction layer is required to avoid duplicating behavior across entry points.

- CLI reuse: one operation surface should serve command workflows without scattered format checks.
- MCP reuse: protocol handlers should call shared capabilities instead of embedding PDF/OFD logic.
- future extensibility: new runtimes (services, workers, agents) should reuse the same contracts.

Abstraction is for reuse and composition, not for erasing format truth.

## Layers

PolarDoc is split into four explicit layers:

- interface layer: `cmd/polardoc`, `cmd/polardoc-mcp`, `internal/mcp`
  - owns user/protocol-facing I/O and transport concerns
  - translates inputs into application use cases
- application layer: `internal/app`
  - composes use cases and orchestration flows
  - does not own format parsing semantics
- abstraction layer: `internal/doc`
  - defines shared capabilities, operation contracts, and narrow interfaces
  - does not define one unified document data model
- format domain layer: `internal/pdf`, `internal/ofd`
  - owns format semantics, parsing models, and format constraints

Dependency direction is one-way:

- interface -> `internal/app` -> `internal/doc` -> format domain implementations
- `internal/pdf` and `internal/ofd` must not depend on each other

## Abstraction Approach

Do not create a unified document model. Use capability-oriented contracts.

### Capabilities

Define capability contracts in `internal/doc`:

- Open
- Validate
- ExtractText
- RenderPreview
- Sign

A format implementation only exposes capabilities it can support reliably.

### Operations

Operations are user-level actions such as annotate, merge, split, and sign.

- each operation has explicit input, output, and failure behavior
- each operation routes to format-aware implementations
- operations must not mutate a fake shared model

### Interfaces

Interfaces are small, explicit, and behavior-first (Go style).
Do not design one interface to cover the full lifecycle.

Conceptual interface roles include:

- Document: a handle for format-specific state and metadata access
- Reader: open/load and read capabilities
- Writer: write/rewrite/incremental output capabilities
- Validator: structural and rule validation capabilities

## Strict Semantic Rule

Abstraction must not break format semantics.

- shared contracts unify intent, not internal structure
- format-specific behavior stays explicit where semantics differ
- if behavior cannot be aligned safely, keep separate format operations

## Anti-Patterns

The following are explicitly disallowed:

- giant `Document` struct attempting to represent every format detail
- shared object graph that flattens PDF and OFD into one pseudo-format
- mixing PDF and OFD logic in the same domain implementation path
- abstraction layers that hide important format constraints from callers

Keep the abstraction layer small, explicit, and honest about differences.
