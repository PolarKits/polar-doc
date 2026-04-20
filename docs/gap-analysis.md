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

### G0 — `README.md` current status is stale
- `README.md` still describes the repository as documentation-only bootstrap code
- current code already includes working CLI commands, PDF and OFD services, and MCP handlers

### G4 — Phase naming is undefined and inconsistent
`NewPhase1Resolver` exists in code. `docs/pdf.md` and `docs/ofd.md` use "Phase-1".
No document defines what Phase-1 means, what Phase-2 would be, or whether
"phase" refers to implementation milestones or something else.

### G7 — Format detection is implicit
Format detection by extension (`.pdf` → `FormatPDF`, `.ofd` → `FormatOFD`) is
hardcoded in `commands/common.go` but not documented as a convention.
If a new format were added, there is no guide for where to register it.

### G8 — `cmd/polardoc-mcp` status is unclear
`cmd/polardoc-mcp/main.go` is empty (placeholder). `docs/mcp.md` describes
a full MCP design. It is unclear whether MCP is in-progress, deferred, or
what subset is currently implemented.

### G9 — contract comments are stale about implemented behavior
- `internal/doc/interfaces.go` still says both PDF and OFD text extraction are stubbed
- `internal/doc/types.go` still says OFD declared version is empty and both extractors are stubs

### G10 — current test and workspace facts are not captured anywhere
- `go test ./...` currently passes
- the worktree already contains unrelated local modifications that new tasks should not overwrite

## Notes

- `extract` JSON support exists in code, so documentation must follow implementation rather than earlier assumptions.
- G0 is the highest-signal mismatch because it misstates the overall state of the repository.
- G7 is descriptive but not urgent without another format in scope.
- G8 reflects a mismatch between current placeholder code and broader MCP documentation.
- G1, G2, G3, G5, G6, and earlier G10 items have been addressed by subsequent documentation work.
- C1 was resolved by earlier doc fixes.
