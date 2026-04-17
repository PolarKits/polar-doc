# Vertical Slice Implementation Guide

A vertical slice in PolarDoc is a capability-oriented feature that spans from CLI command to format implementation, without sharing internal state across PDF and OFD.

## Slice Anatomy

Each slice follows the same layered path:

```
cmd/polardoc/commands/      →  internal/app/  →  internal/doc/  →  internal/pdf|internal/ofd/
     RunXxx                       wiring         interfaces        format implementation
```

## Step-by-Step

### 1. Define Capability Contract

Add the capability interface to `internal/doc/interfaces.go` if it does not already exist.

```go
type Xxxer interface {
    Xxx(ctx context.Context, d Document, req XxxRequest) (XxxResult, error)
}
```

Add corresponding result and request types to `internal/doc/types.go`.

### 2. Implement in Format Packages

In `internal/pdf/service.go` and `internal/ofd/service.go`, add the method to the `Service` implementation.

```go
func (s *service) Xxx(ctx context.Context, d doc.Document, req doc.XxxRequest) (doc.XxxResult, error) {
    pdfDoc, ok := d.(*document)
    if !ok {
        return doc.XxxResult{}, fmt.Errorf("unsupported document type %T", d)
    }
    // format-specific logic
}
```

### 3. Add Command Handler

Create `cmd/polardoc/commands/xxx.go`:

```go
func RunXxx(ctx context.Context, resolver app.ServiceResolver, args []string) error {
    input, err := parseCommandInput("xxx", args)
    if err != nil {
        return err
    }

    format, err := detectFormatByExtension(input.path)
    if err != nil {
        return err
    }

    ref := doc.DocumentRef{Format: format, Path: input.path}
    svc, ok := resolver.ByFormat(ref.Format)
    if !ok {
        return fmt.Errorf("no service for format %q", ref.Format)
    }

    d, err := svc.Open(ctx, ref)
    if err != nil {
        return err
    }
    defer d.Close()

    result, err := svc.Xxx(ctx, d, req)
    if err != nil {
        return err
    }

    // output formatting (text or JSON)
}
```

### 4. Wire in Root Command

Update `cmd/polardoc/root.go` to register the new subcommand.

## Rules

- Keep PDF and OFD semantics separate; never mix logic in shared code paths.
- Do not introduce a giant unified document model.
- Capability interfaces belong in `internal/doc`; format logic stays in `internal/pdf` and `internal/ofd`.
- Commands follow `RunXxx(ctx context.Context, resolver app.ServiceResolver, args []string) error`.
- Use `writeJSON` for `--json` output; use `fmt.Printf` for text output.
- Return typed errors from commands; let the root command handle exit codes.

## Existing Slices

Refer to `info` and `validate` as reference implementations:

- `cmd/polardoc/commands/info.go` and `cmd/polardoc/commands/validate.go`
- `internal/pdf/service.go` and `internal/ofd/service.go`
