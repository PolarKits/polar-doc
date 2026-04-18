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

## Examples

```bash
polardoc info ./testdata/sample.pdf
polardoc info ./testdata/sample.ofd
polardoc validate ./testdata/sample.pdf
polardoc validate ./testdata/sample.ofd
polardoc extract ./testdata/sample.ofd
```

Use the same command shape for both formats.

## JSON Output Rules

`info` and `validate` support `--json`.

`extract` emits plain text to stdout; `--json` is not supported.

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

**OFD:** Not implemented. Returns exit code 1 with error message `text extraction is not implemented for OFD`.

- use a stable top-level schema per command
- return machine-readable status and errors
- do not mix human-readable prose with JSON output
