# Current Status

## Scope

PolarDoc currently focuses on explicit PDF and OFD document capabilities in a Go codebase with CLI-first execution and early MCP adapters.

## Delivered Now

- CLI command routing for `info`, `validate`, `extract`, and `cp`
- shared service resolution in `internal/app`
- capability contracts in `internal/doc`
- PDF phase-1 read-oriented capabilities:
  - open
  - basic info
  - minimal validation
  - first-page inspection
  - partial text extraction
  - copy/save to a destination path
- OFD phase-1 read-oriented capabilities:
  - open package
  - read version metadata (from OFD.xml Version attribute)
  - count pages (from Document.xml Page elements)
  - validate package structure
  - extract text (TextCode elements across all pages via Content.xml)
- MCP handlers implemented in-process:
  - `pdf_first_page_info`
  - `document_info`

## Recently Delivered (Post Stage 1-9)

- PDF write pipeline: `RewriteFile` — normalizes incremental PDFs to single-revision output
- PDF metadata: `parsePDFName` delimiter fix enabling correct Info dict key parsing
- PDF xref: Prev chain traversal for incremental/linearized PDFs
- PDF page count: `ReadPageCount` reads /Count from root /Pages dict; now populated in `Info`
- OFD text extraction: full implementation traversing Document.xml → page Content.xml → TextCode elements
- OFD sample registry: `internal/testdata/ofd_samples.go` mirrors PDF fixture registry
- CLI: real OFD extraction tests (hello-world, keyword-search, JSON output)

## Deferred

- full MCP server runtime in `cmd/polardoc-mcp` (currently JSON-over-stdin/stdout only)
- deep PDF validation (header presence only; no xref/object integrity)
- full-document PDF text extraction (currently first-page only)
- OFD preview and first-page inspection
- rendering implementations in `internal/render`
- signing and trust implementations in `internal/security`

## Working Facts

- `go test ./...` passes (as of 2026-04-21)
- worktree has uncommitted changes: docs/cli.md (reviewed but not committed), docs/current-status.md (this file)
- git status shows "main...origin/main [ahead 1]" due to stale local tracking ref; actual remote main HEAD is 095c3a4 (same as local HEAD)

## Stage 1-9 Completion Summary (2026-04-20 Evening)

### Stages Delivered

**Stage 1-2**: Core infrastructure — format detection, service resolver, InfoResult fields (Title, Author, Creator, Producer, FileIdentifiers), shared contracts

**Stage 3-4**: PDF read pipeline — open, first-page info, Info metadata, FileIdentifiers from trailer

**Stage 5**: PDF read pipeline hardening — edge cases, error paths, fixture validation

**Stage 6**: OFD implementation — open, version extraction, page count, validation

**Stage 7**: PDF write path — CopyFile implementation, edge case tests, Phase-1/2 documentation note

**Stage 8**: OFD capability completion — Info alignment review, text extraction decision (Phase-2 scope)

**Stage 9**: MCP reality alignment — both handlers registered (pdf_first_page_info, document_info), protocol documentation updated

### Test Status

- `go test ./...` — **PASS** (all packages)
- 14 OFD tests PASS
- 90+ PDF tests PASS  
- MCP handler tests PASS

### Incomplete Items (Next Actions)

1. **PDF xref fixture generation**: BuildMinimalPDF helpers fail with xref offset issues. Next action: Debug buf.Len() tracking during PDF construction, or use external valid PDF fixtures from testdata/pdf/

2. **OFD text extraction**: ~~Declared Phase-2~~ **DELIVERED** (TextCode extraction implemented). Remaining boundaries: text extraction limited to TextCode elements only; complex layouts and other text objects not processed

3. **MCP official protocol**: Currently JSON-over-stdin/stdout only. Next action: Implement MCP server lifecycle and discovery if official MCP support is needed

### Technical Risks for Phase-2

| Risk | Impact |解除条件 |
|------|--------|----------|
| PDF fixture generation with correct xref offsets | Tests cannot validate edge cases | Debug builder.Len() tracking, or use existing valid PDFs |
| ~~OFD text extraction complexity~~ | ~~Cannot extract text for search/analytics~~ | ~~IMPLEMENTED~~ — TextCode extraction delivered; complex text objects remain future work |
| MCP transport layer not implemented | MCP server unusable by official clients | Implement MCP protocol spec or document JSON-over-stdin limitation |
| PDF incremental update writer pipeline | Cannot preserve incremental updates | Design writer pipeline architecture first |

### Files Modified Tonight (git log --name-only HEAD~15..HEAD)

cmd/polardoc/commands/common.go
cmd/polardoc/commands/copy_test.go
cmd/polardoc/commands/e2e_pdf_test.go
cmd/polardoc/commands/extract_test.go
cmd/polardoc/commands/info.go
cmd/polardoc/commands/info_test.go
cmd/polardoc/commands/pdf_samples_test.go
cmd/polardoc-mcp/main.go
cmd/polardoc/root.go
docs/architecture.md
docs/cli.md
docs/contracts/cli-contract.md
docs/current-status.md
docs/gap-analysis.md
docs/mcp.md
docs/progress-tonight-100-tasks.md
internal/app/services.go
internal/doc/format.go
internal/doc/interfaces.go
internal/doc/types.go
internal/mcp/handler.go
internal/mcp/handler_test.go
internal/mcp/pdf_samples_test.go
internal/pdf/pdf_samples_test.go
internal/pdf/realsample_test.go
internal/pdf/service.go
internal/pdf/service_test.go
internal/pdf/write_test.go
internal/testdata/pdf_samples.go
README.md
testdata/ofd/test_core_helloworld.ofd
testdata/ofd/test_core_multipage.ofd
testdata/ofd/test_feat_attachment.ofd
testdata/ofd/test_feat_complex_layout.ofd
testdata/ofd/test_feat_images.ofd
testdata/ofd/test_feat_invoice.ofd
testdata/ofd/test_feat_keyword_search.ofd
testdata/ofd/test_feat_pattern.ofd
testdata/ofd/test_feat_signature.ofd
testdata/ofd/test_feat_transparency.ofd
testdata/pdf/pdf20-with-attachment.pdf
testdata/pdf/pdf-ua-sample.pdf
testdata/pdf/test_core_latex_standard_v1.5.pdf
testdata/pdf/test_core_minimal_v1.5.pdf
testdata/pdf/test_core_multicolumn_v1.5.pdf
testdata/pdf/test_core_multipage_v1.3.pdf
testdata/pdf/test_err_corrupted.pdf
testdata/pdf/test_feat_acroform_v1.5.pdf
testdata/pdf/test_feat_attachment_v1.5.pdf
testdata/pdf/test_feat_cmyk_v1.4.pdf
testdata/pdf/test_feat_complex_attachments_v1.5.pdf
testdata/pdf/test_feat_encrypted_v1.5.pdf
testdata/pdf/test_feat_fillable_v1.6.pdf
testdata/pdf/test_feat_image_mask.pdf
testdata/pdf/test_feat_layers_v1.5.pdf
testdata/pdf/test_feat_links_v1.5.pdf
testdata/pdf/test_feat_rtl_arabic_v1.5.pdf
testdata/pdf/test_feat_signed_v1.7.pdf
testdata/pdf/test_feat_tagged_v1.7.pdf
testdata/pdf/test_feat_transparency_v1.4.pdf
testdata/pdf/test_perf_large_doc_v1.4.pdf
testdata/pdf/test_std_pdf20_basic.pdf
testdata/pdf/test_std_pdf20_image_bpc.pdf
testdata/pdf/test_std_pdf20_incremental.pdf
testdata/pdf/test_std_pdf20_offset_v2.0.pdf
testdata/pdf/test_std_pdf20_output_intent.pdf
testdata/pdf/test_std_pdf20_utf8_annotation.pdf
testdata/pdf/test_std_pdf20_utf8_v2.0.pdf
testdata/pdf/test_std_pdf20_v2.0.pdf
testdata/pdf/test_std_pdfa2b_v1.7.pdf
testdata/pdf/test_std_pdfa_archival_v1.4.pdf
testdata/pdf/test_ver_compat_v1.0.pdf
testdata/pdf/test_ver_compat_v1.4.pdf
testdata/pdf/test_ver_compat_v1.7.pdf
testdata/pdf/test_ver_v1.0.pdf
testdata/pdf/test_ver_v1.1.pdf
testdata/pdf/with-table.pdf
