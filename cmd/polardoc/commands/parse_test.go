package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarKits/polar-doc/internal/app"
)

func TestParseInfoFileFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{"--file", path}); err != nil {
			t.Fatalf("run info --file: %v", err)
		}
	})

	mustContain(t, output, "format: pdf")
	mustContain(t, output, "path: "+path)
}

func TestParseInfoFFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{"-f", path}); err != nil {
			t.Fatalf("run info -f: %v", err)
		}
	})

	mustContain(t, output, "format: pdf")
	mustContain(t, output, "path: "+path)
}

func TestParseInfoJSONFileFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{"--json", "--file", path}); err != nil {
			t.Fatalf("run info --json --file: %v", err)
		}
	})

	var got struct {
		Format string `json:"format"`
		Path   string `json:"path"`
	}
	mustUnmarshalJSON(t, output, &got)

	if got.Format != "pdf" {
		t.Fatalf("format = %q, want %q", got.Format, "pdf")
	}
	if got.Path != path {
		t.Fatalf("path = %q, want %q", got.Path, path)
	}
}

func TestParseValidateFileFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunValidate(context.Background(), resolver, []string{"--file", path}); err != nil {
			t.Fatalf("run validate --file: %v", err)
		}
	})

	mustContain(t, output, "valid: true")
}

func TestParseValidateFFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunValidate(context.Background(), resolver, []string{"-f", path}); err != nil {
			t.Fatalf("run validate -f: %v", err)
		}
	})

	mustContain(t, output, "valid: true")
}

func TestParseValidateJSONFileFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunValidate(context.Background(), resolver, []string{"--json", "--file", path}); err != nil {
			t.Fatalf("run validate --json --file: %v", err)
		}
	})

	var got struct {
		Valid  bool     `json:"valid"`
		Errors []string `json:"errors"`
	}
	mustUnmarshalJSON(t, output, &got)

	if !got.Valid {
		t.Fatalf("valid = %t, want true", got.Valid)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("errors = %v, want empty", got.Errors)
	}
}

func TestParseMissingPath(t *testing.T) {
	resolver := app.NewPhase1Resolver()

	err := RunInfo(context.Background(), resolver, []string{})
	if err == nil {
		t.Fatalf("run info with no args: expected error, got nil")
	}
}

func TestParseFileWithExtraArg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()

	err := RunInfo(context.Background(), resolver, []string{"--file", path, "extra"})
	if err == nil {
		t.Fatalf("run info --file with extra arg: expected error, got nil")
	}
}

func TestParseTooManyPositionalArgs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()

	err := RunInfo(context.Background(), resolver, []string{path, "extra"})
	if err == nil {
		t.Fatalf("run info with extra positional: expected error, got nil")
	}
}

func TestParseValidateMissingPath(t *testing.T) {
	resolver := app.NewPhase1Resolver()

	err := RunValidate(context.Background(), resolver, []string{})
	if err == nil {
		t.Fatalf("run validate with no args: expected error, got nil")
	}
}

func TestParseValidateFileWithExtraArg(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()

	err := RunValidate(context.Background(), resolver, []string{"--file", path, "extra"})
	if err == nil {
		t.Fatalf("run validate --file with extra arg: expected error, got nil")
	}
}
