# CLI Contract

This documents the current behavior of the PolarDoc CLI.

## Supported Commands

- `info` - print document metadata
- `validate` - check document structural validity

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
