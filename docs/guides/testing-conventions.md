# Testing Conventions

## Command-Layer Tests

Command tests live in `cmd/polardoc/commands/<name>_test.go`.

Each test:
- Creates a temporary directory with `t.TempDir()`
- Writes synthetic fixture content for the document format
- Calls `RunXxx(ctx, resolver, args)` directly
- Captures stdout with `captureStdout(t, func() { ... })`
- Verifies output with `mustContain` or `mustUnmarshalJSON`

Example pattern:
```go
func TestRunInfoPDF(t *testing.T) {
    dir := t.TempDir()
    path := filepath.Join(dir, "sample.pdf")
    if err := os.WriteFile(path, []byte("%PDF-1.4\n"), 0o644); err != nil {
        t.Fatalf("write sample PDF: %v", err)
    }

    resolver := app.NewPhase1Resolver()
    output := captureStdout(t, func() {
        if err := RunInfo(context.Background(), resolver, []string{path}); err != nil {
            t.Fatalf("run info PDF: %v", err)
        }
    })

    mustContain(t, output, "format: pdf")
}
```

## Vertical-Slice Behavior Tests

Vertical-slice tests verify end-to-end behavior from command entry through service layer.

- Test the `RunXxx` command handler, not internal service methods directly.
- Use the same `app.NewPhase1Resolver()` wiring that production uses.
- Do not mock the service layer; use the real format implementations.

## stdout/stderr Separation

The `captureStdout` helper replaces `os.Stdout` with a pipe during test execution.

- stdout is captured and returned as a string for verification.
- stderr is not captured; errors printed to stderr via `fmt.Fprintln(stderr, ...)` are observed through `t.Fatalf` after the command returns.

```go
func captureStdout(t *testing.T, run func()) string {
    t.Helper()
    oldStdout := os.Stdout
    r, w, err := os.Pipe()
    // ...
    os.Stdout = w
    run()
    _ = w.Close()
    os.Stdout = oldStdout
    // ...
}
```

## JSON Output Verification

JSON output is verified by unmarshaling into a typed struct:

```go
var got struct {
    Format string `json:"format"`
    Path   string `json:"path"`
}
mustUnmarshalJSON(t, output, &got)
if got.Format != "pdf" {
    t.Fatalf("format = %q, want %q", got.Format, "pdf")
}
```

Helper functions:
- `mustUnmarshalJSON(t, output, dst)` - fails if output is not valid JSON
- `mustUnmarshalValidateJSON(t, output, dst)` - same, used for validate command output

## Runtime Error vs Invalid-Document Behavior

These are distinct outcomes:

| Scenario | Behavior |
|----------|----------|
| Runtime error (file not found, unsupported extension) | Command returns an error immediately; no output printed |
| Validation finds structural issues | Output is printed, then `ErrValidationFailed` is returned as error |

```go
// Runtime error - err is returned directly
if err := RunInfo(ctx, resolver, []string{}); err == nil {
    t.Fatal("expected error")
}

// Invalid document - ErrValidationFailed is returned after output
var runErr error
output := captureStdout(t, func() {
    runErr = RunValidate(ctx, resolver, []string{path})
})
if !errors.Is(runErr, ErrValidationFailed) {
    t.Fatalf("error = %v, want %v", runErr, ErrValidationFailed)
}
```

## Synthetic Fixtures via `t.TempDir()`

Command-layer tests use `t.TempDir()` to create isolated temporary directories.

- Each test is fully self-contained.
- No shared state between tests.
- Fixtures are created inline using `os.WriteFile` with minimal valid content.

```go
dir := t.TempDir()
path := filepath.Join(dir, "sample.pdf")
if err := os.WriteFile(path, []byte("%PDF-1.4\n"), 0o644); err != nil {
    t.Fatalf("write sample PDF: %v", err)
}
```

## Stable Fixtures in `testdata/`

The `testdata/` directories (`testdata/pdf/`, `testdata/ofd/`) currently contain only `.gitkeep` files.

Use stable fixtures from `testdata/` when:
- A test requires a real, complex document that cannot be synthesized inline
- The fixture is version-controlled and shared across tests

For now, prefer inline synthetic fixtures. Reserve `testdata/` for future real-world document fixtures.

## Package Boundaries

Tests stay within their package:

- `cmd/polardoc/commands/*_test.go` - tests command handlers only
- `internal/pdf/*_test.go` - tests PDF service implementation only
- `internal/ofd/*_test.go` - tests OFD service implementation only

Cross-package integration is tested at the command layer only.
