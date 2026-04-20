# Progress And Tonight 100 Tasks

## Current Progress Check

This repository is no longer a pure bootstrap skeleton.
The current implementation status is closer to "phase-1 partial delivery with passing tests".

### What is already working

- CLI entry `cmd/polardoc` exists and routes `info`, `validate`, `extract`, and `cp`
- `go test ./...` passes in the current workspace
- `internal/app` resolves PDF and OFD services through explicit format routing
- `internal/pdf` can:
  - open PDF files
  - read declared header version
  - return basic info
  - read trailer `/ID`
  - read Info dictionary metadata such as title and author
  - perform minimal validation
  - inspect first-page structure
  - extract some text from first-page content streams
  - copy a PDF to a new output path
- `internal/ofd` can:
  - open OFD packages
  - read package size
  - parse basic version metadata from `OFD.xml`
  - count pages
  - perform basic package validation
- `internal/mcp` already contains two read handlers:
  - `pdf_first_page_info`
  - `document_info`

### What is still missing or weak

- `cmd/polardoc-mcp` is still placeholder-level and not a real server entrypoint
- PDF validation is still shallow; it does not yet verify full xref/object/trailer integrity
- PDF extraction is limited and not yet robust across encodings, operators, and multi-page traversal
- PDF write support is only copy/save, not a clean writer pipeline
- OFD extraction, preview, and first-page inspection are still not implemented
- `internal/render` and `internal/security` are interface placeholders only
- some docs are stale or internally inconsistent with current code

### Main factual gaps found

- `README.md` still says the repo is "documentation only", which is no longer true
- `internal/doc` comments still say PDF and OFD text extraction are stubs, but PDF now extracts text
- "Phase-1" is used in code and docs without a strict phase definition
- format detection by extension is implemented in multiple places and not documented as one shared rule
- MCP docs describe a broader surface than the actual executable entrypoint currently provides

## Tonight Execution Rule

Execute tasks strictly in numeric order.
Do not start MCP write flows, render work, or security work before the contract and documentation cleanup tasks are done.
Do not flatten PDF and OFD semantics into one document model.

## 100 Sequential Tasks For Tonight

### Stage 1 — Freeze Facts And Clean Contracts

1. Record the current `go test ./...` result in a progress note.
2. Record current dirty files from `git status --short` so tonight's work does not overwrite them.
3. Update `README.md` current-status section to reflect that the project already has partial implementation.
4. Update `README.md` repository overview to mention the four implemented CLI commands.
5. Update `README.md` to state that MCP handlers exist but MCP server entrypoint is incomplete.
6. Define "Phase-1" explicitly in one architecture document.
7. Define the expected next milestone name after Phase-1 to remove ambiguity.
8. Update `docs/gap-analysis.md` with the latest factual status after code inspection.
9. Add one short document that states current scope, delivered scope, and deferred scope.
10. Fix stale comments in `internal/doc/interfaces.go` about text extraction being stubbed for PDF.

### Stage 2 — Unify CLI And Capability Contracts

11. Review `cmd/polardoc/root.go` usage text against the real command set.
12. Review `docs/cli.md` against actual command behavior for `info`, `validate`, `extract`, and `cp`.
13. Document whether `extract --json` is supported and align doc and code if inconsistent.
14. Document `cp` command behavior and limits in `docs/cli.md`.
15. Document `info --page` as PDF-only behavior in `docs/cli.md`.
16. Document format detection by extension as the current routing rule.
17. Decide whether uppercase extensions are supported and document the behavior.
18. Remove duplicate extension-detection logic or centralize it behind one helper.
19. Review JSON output schemas across commands for field naming consistency.
20. Document command exit-code behavior for all implemented commands.

### Stage 3 — Harden CLI Tests

21. Audit command tests for coverage of success and error paths.
22. Add a test for `info --page` on non-PDF input.
23. Add a test for `cp` with unsupported OFD input.
24. Add a test for `extract --json` success path on a known-good PDF.
25. Add a test for `extract --json` error payload on unsupported extension input.
26. Add a test for `validate --json` returning `ErrValidationFailed` on invalid input.
27. Add a test for CLI usage text on unknown command.
28. Add a test for missing positional path on each command.
29. Add a test matrix for uppercase `.PDF` and `.OFD` inputs.
30. Re-run command package tests and stabilize snapshots if output changed.

### Stage 4 — Stabilize Shared Types And Service Boundaries

31. Review `internal/app` interfaces for methods that are truly used by CLI and MCP today.
32. Remove unused capability expectations from current phase-1 service interfaces if they create false promises.
33. Ensure `internal/app` remains wiring-only and does not gain format logic.
34. Review `internal/doc/types.go` comments for phase-1 vs current implementation accuracy.
35. Clarify which `InfoResult` fields are currently populated by PDF and OFD.
36. Clarify which `ValidationReport` guarantees are structural only.
37. Clarify the actual semantics of `TextResult` ordering and completeness.
38. Review `FirstPageInfoResult` as a capability contract, not a fake shared model.
39. Confirm no package imports violate PDF/OFD separation.
40. Add one architecture note on how new formats should be registered without cross-domain leakage.

### Stage 5 — PDF Read Pipeline Hardening

41. Audit PDF open/info/validate path for seek/reset correctness.
42. Add tests for malformed PDF header behavior.
43. Add tests for missing `startxref` behavior in `Info`.
44. Add tests for malformed trailer `/ID` parsing.
45. Add tests for malformed Info dictionary references.
46. Add tests for PDFs without metadata but with valid core structure.
47. Review first-page parser failure modes for corrupted xref cases.
48. Add a targeted test for inline page resources handling.
49. Add a targeted test for page rotation extraction.
50. Add a targeted test for multi-content-stream first-page parsing.

### Stage 6 — PDF Text Extraction Improvement

51. Review current extraction algorithm and list supported operators and known unsupported operators.
52. Add a test for literal string extraction with escaped parentheses.
53. Add a test for hex-string text content if support is intended.
54. Add a test for compressed content streams already covered by zlib decode.
55. Add a test for PDFs with multiple text fragments on one page.
56. Decide whether extraction is first-page-only or document-wide for Phase-1.
57. If first-page-only, document that limit clearly in CLI and capability comments.
58. If document-wide is desired, add page traversal support without changing the contract shape.
59. Improve extraction error messages so they distinguish "unsupported" from "corrupt input".
60. Re-run PDF tests on all sample files in `testdata/pdf`.

### Stage 7 — PDF Write And Save Path

61. Review `cp` command semantics and decide whether it is raw copy or normalized save.
62. Document the exact current behavior of PDF save/copy.
63. Add a test that verifies output bytes match input bytes for current copy behavior.
64. Add a test for destination overwrite behavior.
65. Add a test for source path not found.
66. Add a test for destination directory missing.
67. Decide whether `cp` should live as `cp` or a more explicit `save` command in the future.
68. If keeping `cp`, align docs and usage text consistently.
69. If planning a writer pipeline, add a doc note separating current copy behavior from future rewrite behavior.
70. Keep PDF write work isolated from OFD write semantics.

### Stage 8 — OFD Minimal Capability Completion

71. Audit OFD `Info` implementation against `docs/ofd.md` minimal phase-1 milestone.
72. Add tests for OFD version extraction from `OFD.xml`.
73. Add tests for OFD page count extraction.
74. Add tests for missing `OFD.xml`.
75. Add tests for missing `Document.xml`.
76. Add tests for invalid `DocRoot`.
77. Decide whether OFD text extraction stays explicitly unsupported for tonight.
78. If unsupported, document the limitation in `docs/ofd.md`, `docs/cli.md`, and contract comments.
79. If minimal extraction is feasible, scope it to a tiny XML text path and not a fake cross-format abstraction.
80. Re-run OFD tests and ensure error messages are stable and intentional.

### Stage 9 — MCP Reality Alignment

81. Document actual MCP status separately from aspirational MCP design.
82. Add one minimal `cmd/polardoc-mcp` entrypoint that can register existing read handlers.
83. Define the exact transport or placeholder startup behavior for that entrypoint.
84. Ensure MCP input validation errors return structured, stable messages.
85. Review `detectFormatByExtension` in `internal/mcp` for short-path panic risk.
86. Fix any extension parsing that assumes path length is at least four bytes.
87. Add tests for uppercase `.PDF` in MCP handlers.
88. Add tests for unsupported extension handling in MCP handlers.
89. Add tests for empty or too-short paths in MCP handlers.
90. Update `docs/mcp.md` to separate "implemented now" from "planned later".

### Stage 10 — Finalize Tonight Deliverables

91. Re-run `go test ./...` after all edits.
92. Review changed docs for contradictions between README, architecture, CLI, PDF, OFD, and MCP docs.
93. Review exported comments for stale phase statements.
94. Verify no task introduced a flattened PDF/OFD model.
95. Verify no new package dependency breaks the architecture rules.
96. Prepare a concise progress summary for the next session.
97. Prepare a concise risk list for what still blocks Phase-2.
98. Prepare a file list of tonight's touched paths for safe review.
99. Split completed work into logical commits instead of one large mixed commit.
100. Mark unfinished items with exact next action, not vague TODO text.

## Recommended Stop Conditions

If time becomes tight, stop after task 30, 60, 80, or 90.
Those are the cleanest checkpoints for a pause because they end after contract cleanup, PDF hardening, OFD hardening, and MCP alignment respectively.

## Priority Judgment

If only one person is executing tonight, the highest-value order is:

- first finish tasks 1 through 20 to make the project state truthful
- then finish tasks 21 through 60 to stabilize the current strongest path, which is PDF via CLI
- then finish tasks 81 through 90 so MCP status stops being misleading
- then use remaining time on tasks 71 through 80 for OFD hardening

This order matches the actual code maturity of the repository.
