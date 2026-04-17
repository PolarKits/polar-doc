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
polardoc extract --text ./testdata/sample.pdf
polardoc extract --text ./testdata/sample.ofd
```

Use the same command shape for both formats.

## JSON Output Rules

All commands should support `--json`.

- use a stable top-level schema per command
- return machine-readable status and errors
- do not mix human-readable prose with JSON output
