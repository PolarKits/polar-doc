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
  - read version metadata
  - count pages
  - validate package structure
- MCP handlers implemented in-process:
  - `pdf_first_page_info`
  - `document_info`

## Deferred

- full MCP server runtime in `cmd/polardoc-mcp`
- deep PDF validation and robust full-document extraction
- PDF rewrite and incremental-update writer pipeline
- OFD text extraction, preview, and first-page inspection
- rendering implementations in `internal/render`
- signing and trust implementations in `internal/security`

## Working Facts

- `go test ./...` passes in the repository state inspected on 2026-04-20
- existing local modifications observed during inspection:
  - `cmd/polardoc/commands/info_test.go`
  - `internal/mcp/handler_test.go`
  - `internal/pdf/service.go`
  - `internal/pdf/service_test.go`
