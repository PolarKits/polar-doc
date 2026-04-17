# PolarDoc

PolarDoc is a Go-based document platform focused on fixed-layout formats, currently PDF and OFD.
It is designed as a foundation for multiple products and runtimes, including a CLI, an MCP server, and future service deployments.

## Positioning

PolarDoc is a platform, not a single parser package.
The repository keeps format-specific semantics explicit while sharing cross-cutting operations through capability-oriented abstractions.

## Scope

- PDF domain support as an independent module space
- OFD domain support as an independent module space
- Shared document capabilities (for example inspection, extraction, validation contracts)
- Multi-entry architecture for CLI and MCP
- Foundations for future service runtimes

## Non-Scope

- A flattened, unified document model that erases PDF/OFD differences
- Early over-abstraction or framework-heavy layering
- Complete format implementation in this bootstrap stage
- UI or end-user office suite features

## Architecture Overview

PolarDoc uses four explicit layers:

- interface layer: `cmd/polardoc`, `cmd/polardoc-mcp`, `internal/mcp`
- application layer: `internal/app`
- abstraction layer: `internal/doc` for capability-oriented contracts
- format domain layer: `internal/pdf`, `internal/ofd`

Rules:

- `internal/pdf` and `internal/ofd` must not depend on each other
- Shared logic must flow through `internal/doc`
- No unified document model across PDF and OFD
- Format-specific details stay in their own domains

`internal/app` composes use cases; interface packages expose CLI and MCP entry points.

## Repository Layout

- `cmd/polardoc`: CLI entry point
- `cmd/polardoc-mcp`: MCP server entry point
- `internal/app`: application assembly and runtime wiring
- `internal/doc`: shared capability contracts across formats
- `internal/pdf`: PDF domain
- `internal/ofd`: OFD domain
- `internal/render`: rendering orchestration and output pipelines
- `internal/security`: signatures, crypto, trust, and policy concerns
- `internal/mcp`: MCP protocol adapters and handlers
- `docs/`: architecture and domain notes
- `testdata/`: stable fixtures for tests and compatibility checks

## Relationship with PolarOffice

PolarDoc is intended to become the document engine layer for the future PolarOffice ecosystem.
PolarOffice should consume PolarDoc as an internal platform dependency, while PolarDoc remains focused on format fidelity, document operations, and protocol-friendly integration surfaces.

## Current Status

This repository is bootstrapped with structure and architectural documentation only.
Implementation packages are intentionally placeholders to preserve clean boundaries from the start.
