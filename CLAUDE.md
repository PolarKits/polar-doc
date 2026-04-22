# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Contributor Policy

Claude must NOT appear as a contributor or co-author in any commit.
Do NOT add `Co-Authored-By: Claude` or any `Co-Authored-By` trailer to commit messages.
All commits must use only the git user identity configured in the repository (janssenkm).

## Build and Test Commands

```bash
go build ./cmd/polardoc      # Build CLI
go build ./cmd/polardoc-mcp  # Build MCP server
go test ./...                 # Run all tests
go test ./cmd/polardoc/...    # Run CLI tests
go test ./internal/pdf/...    # Run PDF service tests
go test ./internal/ofd/...    # Run OFD service tests
```

## Architecture

PolarDoc is a Go document platform supporting PDF and OFD formats. It uses a four-layer architecture:

```
cmd/polardoc, cmd/polardoc-mcp  (interface layer)
                ↓
         internal/app            (application layer: wires services)
                ↓
         internal/doc            (abstraction layer: capability contracts)
                ↓
    internal/pdf | internal/ofd (format domain layer)
```

**Critical rule**: `internal/pdf` and `internal/ofd` must NOT depend on each other. All shared logic flows through `internal/doc`.

## Capability Contracts (internal/doc)

Interfaces are narrow and behavior-oriented:
- `Document` — runtime handle with `Ref()` and `Close()`
- `Opener` — opens a document from a `DocumentRef`
- `InfoProvider` — returns `InfoResult`
- `Validator` — returns `ValidationReport`
- `TextExtractor` — returns `TextResult`
- `PreviewRenderer` — returns `PreviewResult`
- `Signer` — optional capability

Format detection is by file extension (`.pdf` → `FormatPDF`, `.ofd` → `FormatOFD`).

## Service Resolution (internal/app)

`NewPhase1Resolver()` wires the application layer. `StaticResolver.ByFormat(format)` dispatches to the appropriate format service (PDF or OFD).

## CLI Commands

- `polardoc info <file>` — show document metadata
- `polardoc validate <file>` — validate document structure
- `polardoc extract --text <file>` — extract text content

All commands support `--json` for machine-readable output. Exit code 1 indicates validation failure or error.

## Repository Layout

```
cmd/polardoc/           CLI entry point and commands
cmd/polardoc-mcp/       MCP server entry point (placeholder)
internal/app/          Service resolution and wiring
internal/doc/          Capability contracts and shared types
internal/pdf/          PDF format implementation
internal/ofd/          OFD format implementation
internal/mcp/          MCP protocol adapters
internal/render/       Rendering interfaces
internal/security/     Signing and crypto interfaces
testdata/pdf/, testdata/ofd/  Stable test fixtures
docs/                  Architecture and domain notes
```
