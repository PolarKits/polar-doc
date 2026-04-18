# CLI Contract

This documents the current behavior of the PolarDoc CLI.

## Supported Commands

- `info` - print document metadata, optionally with `--page` for first page info
- `validate` - check document structural validity
- `extract` - extract text from document

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

## Help and Usage

- Running `polardoc help` (or `-h`, `--help`) prints the usage line and exits 0.
- Running `polardoc` with no arguments prints usage and exits 1.
- Running `polardoc <unknown>` prints an error with usage and exits 1.
