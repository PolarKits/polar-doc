# CLI Contract

This documents the current behavior of the PolarDoc CLI.

## Supported Commands

- `info` - print document metadata, optionally with `--page` for first page info
- `validate` - check document structural validity
- `extract` - extract text from document
- `cp` - copy a PDF document to a destination path

## Path Input

All commands accept a document path via one of:

- positional argument: `polardoc info document.pdf`
- `--file <path>`: `polardoc info --file document.pdf`
- `-f <path>`: `polardoc info -f document.pdf`

Only `.pdf` and `.ofd` extensions are accepted.

## Output Modes

### Plain Text

**info:**
```
format: <pdf|ofd>
path: <string>
size_bytes: <integer>
declared_version: <string>   # only if non-empty
```

**info --page (PDF only):**
```
path: <string>
pages_ref: <obj_num> <gen_num> R
page_ref: <obj_num> <gen_num> R
parent: <obj_num> <gen_num> R
media_box: [<float>, <float>, <float>, <float>]
resources: <obj_num> <gen_num> R
contents: <obj_num> <gen_num> R[, ...]
rotate: <integer>   # only if non-zero
```

**validate:**
```
valid: <true|false>
error: <string>   # repeated once per error
```

**cp (PDF only):**
```
copied <src> to <dst>
```

### JSON

**info:**
```json
{
  "format": "<pdf|ofd>",
  "path": "<string>",
  "size_bytes": <integer>,
  "declared_version": "<string>"   // omitted if empty
}
```

**info --page (PDF only):**
```json
{
  "path": "<string>",
  "pages_ref": {"obj_num": <int>, "gen_num": <int>},
  "page_ref": {"obj_num": <int>, "gen_num": <int>},
  "parent": {"obj_num": <int>, "gen_num": <int>},
  "media_box": [<float>, <float>, <float>, <float>],
  "resources": {"obj_num": <int>, "gen_num": <int>},
  "contents": [{"obj_num": <int>, "gen_num": <int>}, ...],
  "rotate": <int>   // omitted if null
}
```

**validate:**
```json
{
  "valid": <boolean>,
  "errors": ["<string>"]
}
```

## Exit Codes

| Scenario | Exit Code |
|----------|-----------|
| Command succeeded | 0 |
| Usage error (missing args, invalid flags, unknown command) | 1 |
| Runtime error (file not found, format error) | 1 |
| Validation found structural issues | 1 |
| Help requested (`help`, `-h`, `--help`) | 0 |

## cp Command

`cp` copies a PDF document to a destination path.

**Usage:** `polardoc cp <src.pdf> <dst.pdf>`

**Behavior:**
- Reads the source PDF and copies its bytes to the destination
- This is a raw file copy, NOT PDF editing (no version upgrade, no metadata modification)
- Copied file can be processed through the normal read pipeline (Open, Validate, FirstPageInfo)
- Only PDF format is supported; OFD returns error

**Error cases:**
- Missing arguments: `usage: polardoc cp <src> <dst>`
- Non-PDF source: `save only supported for PDF`
- Source file not found: standard file system error

## extract Command

`extract` extracts text from a document.

**Usage:** `polardoc extract <input>`

**Behavior:**
- PDF: not implemented, returns error `text extraction is not implemented for PDF`
- OFD: not implemented, returns error `text extraction is not implemented for OFD`
- `--json` flag is not supported

**Exit codes:**
- 1: all extract calls (not implemented) or usage error
- 0: never (extract is not implemented for any format)

**Note:** testdata/ofd contains no real OFD samples.

## Help and Usage

- Running `polardoc help` (or `-h`, `--help`) prints the usage line and exits 0.
- Running `polardoc` with no arguments prints usage and exits 1.
- Running `polardoc <unknown>` prints an error with usage and exits 1.
