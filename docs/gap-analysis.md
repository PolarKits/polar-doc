# Gap Analysis — Phase 0

This document records factual gaps between current documentation and current code.
It is not a roadmap or design document.

## Current Documentation Inventory

| File | Covers |
|------|--------|
| `README.md` | Positioning, scope, 4-layer architecture, repo layout |
| `docs/architecture.md` | Why abstraction, layers, capability contracts, anti-patterns |
| `docs/architecture-rules.md` | Derived rules: package boundaries, no flattening, wiring-only app |
| `docs/code-structure.md` | Package responsibilities, dependency rules, interfaces philosophy |
| `docs/cli.md` | CLI philosophy, command names, examples, JSON output rule |
| `docs/contracts/cli-contract.md` | Exit-code semantics, output format stability, ErrValidationFailed |
| `docs/pdf.md` | PDF domain responsibilities, components, version policy, phase-1 goals |
| `docs/ofd.md` | OFD domain responsibilities, components, differences from PDF, phase-1 goals |
| `docs/mcp.md` | MCP purpose, read/write separation, preview/commit flow, safety constraints |
| `docs/guides/testing-conventions.md` | Command-layer test patterns, stdout/stderr capture, JSON verification, fixture strategy |
| `docs/guides/vertical-slice-guide.md` | Adding a new command: resolver wiring, contract interfaces, testing |
| `AGENTS.md` | Minimal repo structure, scope, no-flattening rule |
| `CLAUDE.md` | Build/test commands, architecture summary, CLI commands, repo layout |

## Gaps

### G0 — `README.md` current status is stale — **Closed / 已关闭** (2026-04-21)
- `README.md` still describes the repository as documentation-only bootstrap code
- current code already includes working CLI commands, PDF and OFD services, and MCP handlers
- **关闭原因**: README.md 当前已有 Current Status 节，正确描述 phase-1 partial-delivery 状态

### G4 — Phase naming is undefined and inconsistent — **Closed / 已关闭** (2026-04-21)
- `NewPhase1Resolver` exists in code, `docs/pdf.md` and `docs/ofd.md` use "Phase-1"
- docs/pdf.md lines 68-77 define "Minimal Phase-1 Milestone (DELIVERED)" with explicit capabilities
- docs/ofd.md lines 50-58 define the same for OFD
- Phase-1 and Phase-2 scope is now documented in docs/pdf.md (current implementation vs future responsibilities)
- **关闭原因**: Phase naming is now consistently documented across docs/pdf.md and docs/ofd.md

### G7 — Format detection is implicit — **Closed / 已关闭** (2026-04-21)
- docs/cli.md lines 84-90 explicitly document format routing rules ("Routing Rules" section)
- `internal/doc/format.go` has doc comment explaining extension-based case-insensitive routing
- **关闭原因**: Format detection is now documented in docs/cli.md "Routing Rules" section

### G8 — `cmd/polardoc-mcp` status is unclear — **Closed / 已关闭** (2026-04-21)
- `cmd/polardoc-mcp/main.go` is NOT empty; it implements a full JSON-over-stdin/stdout runtime with three MCP handlers: `pdf_first_page_info`, `document_info`, and `document_validate`
- docs/mcp.md lines 3-7 explicitly state "Current Implementation: JSON-over-stdin/stdout... does NOT use the official MCP protocol spec"
- docs/current-status.md line 41 lists it under "Deferred: full MCP server runtime in cmd/polardoc-mcp (currently JSON-over-stdin/stdout only)"
- **关闭原因**: cmd/polardoc-mcp/main.go is implemented (74 lines), but official MCP protocol remains unimplemented; current status is correctly documented in docs/mcp.md and docs/current-status.md

### G9 — contract comments are stale about implemented behavior — **Closed / 已关闭** (2026-04-21)
- `internal/doc/interfaces.go` still says both PDF and OFD text extraction are stubbed
- `internal/doc/types.go` still says OFD declared version is empty and both extractors are stubs
- **关闭原因**: `internal/doc/interfaces.go` 中 TextExtractor 注释和 `internal/doc/types.go` 中 TextResult 注释已正确描述 OFD 文本提取为已实现

### G10 — current test and workspace facts are not captured anywhere — **Closed / 已关闭** (2026-04-24)
- `go test ./...` passes as of 2026-04-24
- worktree is clean (no uncommitted changes)
- **关闭原因**: Current state is accurately described in docs/current-status.md as of 2026-04-24. This item is closed as the worktree has stabilized.

## Notes

- G0, G4, G7, G8, G9 are Closed as of 2026-04-21.
- G10 is Closed as of 2026-04-24 — see G10 entry for rationale.
- G1, G2, G3, G5, G6, and earlier G10 items were addressed by prior documentation work.
- C1 was resolved by earlier doc fixes.
