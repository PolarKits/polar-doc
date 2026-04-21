# CLI

## CLI Philosophy

The PolarDoc CLI follows Unix-style principles:

- one command, one clear task
- explicit input and explicit output
- predictable behavior for scripting and automation

The CLI exposes capabilities while keeping PDF and OFD semantics explicit.

## Command Shape

`polardoc <command> [flags] <input>`

Core commands:

- `info`
- `validate`
- `extract`
- `cp`

## Examples

```bash
polardoc info ./testdata/sample.pdf
polardoc info ./testdata/sample.ofd
polardoc validate ./testdata/sample.pdf
polardoc validate ./testdata/sample.ofd
polardoc extract ./testdata/sample.pdf
polardoc cp ./testdata/sample.pdf ./tmp/copied.pdf
```

Use the same command shape for both formats.

## JSON Output Rules

`info`, `validate`, and `extract` support `--json`.

`cp` emits plain text only.

## Extract Command Behavior

**PDF:** Minimal text extraction is implemented for some PDFs. Returns non-empty text for PDFs with literal string content or valid FlateDecode streams. Returns an error for PDFs with:
- Corrupted content streams (e.g., zlib decompression failures)
- XRef corruption (unable to read document structure)

Current compatibility (testdata/pdf):
| Sample | Result |
|--------|--------|
| pdf20-utf8-test.pdf | ✓ Non-empty text |
| sample-local-pdf.pdf | ✓ Non-empty text |
| testPDF_Version.5.x.pdf | ✓ Non-empty text |
| Red_Hat_OpenShift_Serverless...pdf | ✗ zlib: invalid header |
| testPDF_Version.8.x.pdf | ✗ XRef: object not found |

**OFD:** Text extraction is implemented. It traverses the Document.xml page list and extracts TextCode elements from each page's Content.xml. Scope and limitations:

- Extracts text from TextCode elements in page content streams
- Multi-page documents are supported; text from each page is joined with newlines
- Text extraction is limited to TextCode elements only; other text objects or complex layouts are not processed
- XML structure is not fully validated; malformed content may be skipped silently

When `--json` is used, `extract` emits:

```json
{
  "text": "..."
}
```

On extract failure with `--json`, the command currently emits:

```json
{
  "error": "..."
}
```

and still exits with code 1.

## Routing Rules

Format routing is currently extension-based and case-insensitive.

- `.pdf` routes to `FormatPDF`
- `.ofd` routes to `FormatOFD`

Any other extension returns an error.

## cp Command Behavior

`cp` is currently PDF-only.
It copies the source PDF bytes to a destination path and does not perform PDF normalization, upgrade, or editing.

> **Phase note:** Current `cp` is raw byte copy (Phase-1). Full PDF editing (content modification, metadata updates, version upgrade) via writer pipeline is Phase-2 work.

- use a stable top-level schema per command
- return machine-readable status and errors
- do not mix human-readable prose with JSON output
