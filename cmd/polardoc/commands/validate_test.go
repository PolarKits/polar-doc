package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarKits/polardoc/internal/app"
)

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
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml": "<ofd/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write bad OFD: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunValidate(context.Background(), resolver, []string{path}); err != nil {
			t.Fatalf("run validate OFD: %v", err)
		}
	})

	mustContain(t, output, "valid: false")
	mustContain(t, output, "error: invalid OFD package: missing Document.xml")
}
