# Architecture Rules

These rules define package boundaries, allowed dependencies, and explicit prohibitions.
If this document conflicts with another doc, this document governs for matters of package boundary and dependency direction.

## Layer Roles

### cmd/polardoc — Interface Layer
- Owns CLI entry point, argument parsing, and output formatting.
- Delegates all document operations to `internal/app` via `app.ServiceResolver`.
- Must not contain format parsing or validation logic.

### internal/app — Application Wiring
- Assembles and routes to concrete format services.
- Holds `StaticResolver.ByFormat` dispatch logic.
- Must not contain format parsing, validation semantics, or cross-domain document logic.

### internal/doc — Capability Contracts
- Defines narrow, behavior-oriented interfaces: `Opener`, `InfoProvider`, `Validator`, `TextExtractor`, `PreviewRenderer`, `Signer`.
- Defines shared types and format string constants (`FormatPDF = "pdf"`, `FormatOFD = "ofd"`).
- Does not define a unified in-memory document model.
- Does not contain concrete format implementations.

### internal/pdf — PDF Domain
- Owns PDF-specific parsing, validation, and format semantics.
- Implements `doc` interfaces for PDF.
- Must not import `internal/ofd`.

### internal/ofd — OFD Domain
- Owns OFD-specific package handling, XML parsing, and format semantics.
- Implements `doc` interfaces for OFD.
- Must not import `internal/pdf`.

## Dependency Rules

- `internal/doc` defines shared capability contracts and types; it must not depend on `internal/pdf` or `internal/ofd`.
- `internal/pdf` and `internal/ofd` may depend on `internal/doc` for capability interfaces and shared types.
- `internal/app` may depend on `internal/doc`, `internal/pdf`, and `internal/ofd` for service wiring; it has no format logic.
- `cmd/polardoc` may depend on `internal/app` and use shared `internal/doc` types where needed.
- `internal/pdf` and `internal/ofd` must not depend on each other.

## Format Detection
- Format is detected by file extension: `.pdf` → `FormatPDF`, `.ofd` → `FormatOFD`.
- Routing is in `cmd/polardoc/commands/common.go`.
- Adding a new format requires an extension entry and service wiring in `internal/app`.

## Allowed Abstractions
- Narrow interfaces in `internal/doc` representing single operations.
- `FormatService` composition and `ServiceResolver` dispatch in `internal/app`.
- Format-specific types within `internal/pdf` or `internal/ofd`.

## Forbidden Patterns
- **No unified document model**: no type representing both PDF and OFD internals.
- **No cross-domain dependency**: `internal/pdf` and `internal/ofd` must not import each other.
- **No format logic in app layer**: `internal/app` must not parse, validate, or inspect document content.
- **No format logic in interface layer**: `cmd/polardoc` must not contain document parsing or validation logic.
- **No implementations in doc layer**: `internal/doc` must not contain concrete format handling code.
- **No premature abstraction**: do not add interfaces for capabilities not yet implemented.

## Concern Placement

| Concern | Location |
|---------|----------|
| CLI/interface concerns | `cmd/polardoc/` |
| Service wiring and routing | `internal/app` |
| Capability interfaces and shared types | `internal/doc` |
| Format-specific semantics | `internal/pdf` or `internal/ofd` |
| Synthetic test fixtures | test package (inline via `t.TempDir()`) |
| Stable test fixtures | `testdata/pdf/`, `testdata/ofd/` |
