package commands

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarKits/polardoc/internal/app"
)

// TestRunValidateInvalidPDF verifies that a corrupted PDF file returns
// valid:false and a specific error message about the PDF being invalid.
func TestRunValidateInvalidPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupted.pdf")
	// Write a file that has no valid PDF header - just random bytes.
	// The PDF service validates by checking for "%PDF-" header, so this
	// should fail validation since it's not a valid PDF file.
	if err := os.WriteFile(path, []byte("not a pdf file at all"), 0o644); err != nil {
		t.Fatalf("write corrupted PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	var runErr error
	output := captureStdout(t, func() {
		runErr = RunValidate(context.Background(), resolver, []string{path})
	})

	// The validation should fail with an error about invalid PDF structure
	if runErr == nil {
		t.Fatal("run validate corrupted PDF: expected error, got nil")
	}
	mustContain(t, output, "valid: false")
	mustContain(t, output, "error:")
}

func TestRunValidatePDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunValidate(context.Background(), resolver, []string{path}); err != nil {
			t.Fatalf("run validate PDF: %v", err)
		}
	})

	mustContain(t, output, "valid: true")
}

func TestRunValidateInvalidOFD(t *testing.T) {
	path := writeInvalidOFD(t)
	resolver := app.NewPhase1Resolver()
	var runErr error
	output := captureStdout(t, func() {
		runErr = RunValidate(context.Background(), resolver, []string{path})
	})

	if !errors.Is(runErr, ErrValidationFailed) {
		t.Fatalf("run validate OFD error = %v, want %v", runErr, ErrValidationFailed)
	}
	mustContain(t, output, "valid: false")
	mustContain(t, output, "error: invalid OFD package: missing Document.xml")
}

func TestRunValidateJSONValidPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunValidate(context.Background(), resolver, []string{"--json", path}); err != nil {
			t.Fatalf("run validate PDF JSON: %v", err)
		}
	})

	var got struct {
		Valid  bool     `json:"valid"`
		Errors []string `json:"errors"`
	}
	mustUnmarshalValidateJSON(t, output, &got)

	if !got.Valid {
		t.Fatalf("valid = %t, want true", got.Valid)
	}
	if len(got.Errors) != 0 {
		t.Fatalf("errors = %v, want empty", got.Errors)
	}
}

func TestRunValidateJSONInvalidOFD(t *testing.T) {
	path := writeInvalidOFD(t)
	resolver := app.NewPhase1Resolver()
	var runErr error
	output := captureStdout(t, func() {
		runErr = RunValidate(context.Background(), resolver, []string{"--json", path})
	})

	if !errors.Is(runErr, ErrValidationFailed) {
		t.Fatalf("run validate OFD JSON error = %v, want %v", runErr, ErrValidationFailed)
	}

	var got struct {
		Valid  bool     `json:"valid"`
		Errors []string `json:"errors"`
	}
	mustUnmarshalValidateJSON(t, output, &got)

	if got.Valid {
		t.Fatalf("valid = %t, want false", got.Valid)
	}
	if len(got.Errors) != 1 || got.Errors[0] != "invalid OFD package: missing Document.xml" {
		t.Fatalf("errors = %v, want missing Document.xml", got.Errors)
	}
}

func writeInvalidOFD(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml": "<ofd/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write bad OFD: %v", err)
	}
	return path
}

func mustUnmarshalValidateJSON(t *testing.T, output string, dst any) {
	t.Helper()
	if err := json.Unmarshal([]byte(output), dst); err != nil {
		t.Fatalf("unmarshal JSON output %q: %v", output, err)
	}
}
