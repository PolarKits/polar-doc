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

### G3 — `extract` command semantics are thin
`docs/cli.md` shows `extract --text` as an example but:
- `extract.go` does not use `--text` flag; it is positional path only
- `extract` output (plain text to stdout) is not documented
- exit-code behavior for extract is not specified

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

## Contradictions

### C1 — `docs/cli.md` example uses `--text` for extract; code does not
`docs/cli.md` shows `polardoc extract --text ./testdata/sample.pdf`.
`extract.go` parses with `parseDocumentRef` which only reads the path
argument and does not recognize `--text`.

## Notes

- G3 is a direct doc/code mismatch in the current repository.
- G7 is descriptive but not urgent without another format in scope.
- G8 reflects a mismatch between current placeholder code and broader MCP documentation.
- C1 is a current doc/code contradiction.
- G1, G2, G5, G6, and G10 have been addressed by subsequent documentation work.
