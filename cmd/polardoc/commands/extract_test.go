package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarKits/polardoc/internal/app"
)

func TestRunExtractPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.4\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	var runErr error
	captureStdout(t, func() {
		runErr = RunExtract(context.Background(), resolver, []string{path})
	})

	if runErr == nil {
		t.Fatal("run extract PDF: expected error, got nil")
	}
	errStr := runErr.Error()
	if !containsString(errStr, "not implemented") && !containsString(errStr, "too small") && !containsString(errStr, "xref") {
		t.Fatalf("error = %q, want contains 'not implemented' or PDF error", errStr)
	}
}

func TestRunExtractOFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	var runErr error
	captureStdout(t, func() {
		runErr = RunExtract(context.Background(), resolver, []string{path})
	})

	if runErr == nil {
		t.Fatal("run extract OFD: expected error, got nil")
	}
	if !containsString(runErr.Error(), "not implemented") {
		t.Fatalf("error = %q, want contains 'not implemented'", runErr.Error())
	}
}

func TestRunExtractMissingFile(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	var runErr error
	captureStdout(t, func() {
		runErr = RunExtract(context.Background(), resolver, []string{"/tmp/missing.pdf"})
	})

	if runErr == nil {
		t.Fatalf("run extract missing file: expected error, got nil")
	}
}

func TestRunExtractUnsupportedExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write sample txt: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	var runErr error
	captureStdout(t, func() {
		runErr = RunExtract(context.Background(), resolver, []string{path})
	})

	if runErr == nil {
		t.Fatalf("run extract unsupported extension: expected error, got nil")
	}
	if !containsString(runErr.Error(), "unsupported file extension") {
		t.Fatalf("error = %q, want contains %q", runErr.Error(), "unsupported file extension")
	}
}

func TestRunExtractWithFileFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.4\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	var runErr error
	captureStdout(t, func() {
		runErr = RunExtract(context.Background(), resolver, []string{"--file", path})
	})

	if runErr == nil {
		t.Fatal("run extract with --file flag: expected error for PDF, got nil")
	}
	errStr := runErr.Error()
	if !containsString(errStr, "not implemented") && !containsString(errStr, "too small") && !containsString(errStr, "xref") {
		t.Fatalf("error = %q, want contains 'not implemented' or PDF error", errStr)
	}
}

func TestRunExtractWithFFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	var runErr error
	captureStdout(t, func() {
		runErr = RunExtract(context.Background(), resolver, []string{"-f", path})
	})

	if runErr == nil {
		t.Fatal("run extract with -f flag: expected error for OFD, got nil")
	}
	if !containsString(runErr.Error(), "not implemented") {
		t.Fatalf("error = %q, want contains 'not implemented'", runErr.Error())
	}
}

func TestRunExtractRealPDFSuccess(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "pdf", "pdf20-utf8-test.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("pdf20-utf8-test.pdf not found")
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		_ = RunExtract(context.Background(), resolver, []string{path})
	})

	if output == "" {
		t.Fatal("expected non-empty text output")
	}
	if !containsString(output, "PDF") && !containsString(output, "Heading") {
		t.Fatalf("expected text content, got: %q", output)
	}
}

func TestRunExtractRealPDFSuccessSampleLocal(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "pdf", "sample-local-pdf.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("sample-local-pdf.pdf not found")
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		_ = RunExtract(context.Background(), resolver, []string{path})
	})

	if output == "" {
		t.Fatal("expected non-empty text output")
	}
	if len(output) < 50 {
		t.Fatalf("expected substantial text content, got only %d chars: %q", len(output), output)
	}
}

func TestRunExtractRealPDFError(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "pdf", "testPDF_Version.8.x.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testPDF_Version.8.x.pdf not found")
	}

	resolver := app.NewPhase1Resolver()
	var runErr error
	captureStdout(t, func() {
		runErr = RunExtract(context.Background(), resolver, []string{path})
	})

	if runErr == nil {
		t.Fatal("run extract PDF: expected error for corrupted PDF, got nil")
	}
	errStr := runErr.Error()
	if !containsString(errStr, "xref") && !containsString(errStr, "object") {
		t.Fatalf("error = %q, want contains 'xref' or 'object'", errStr)
	}
}

func TestRunExtractRealPDFSuccess5x(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "pdf", "testPDF_Version.5.x.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testPDF_Version.5.x.pdf not found")
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		_ = RunExtract(context.Background(), resolver, []string{path})
	})

	if output == "" {
		t.Fatal("expected non-empty text output")
	}
	if !containsString(output, "PDF") && !containsString(output, "Version") {
		t.Fatalf("expected text content, got: %q", output)
	}
}

func TestRunExtractRealPDFErrorRedHat(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "pdf", "Red_Hat_OpenShift_Serverless-1.35-Serverless_Logic-en-US.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("Red_Hat_OpenShift_Serverless-1.35-Serverless_Logic-en-US.pdf not found")
	}

	resolver := app.NewPhase1Resolver()
	var runErr error
	captureStdout(t, func() {
		runErr = RunExtract(context.Background(), resolver, []string{path})
	})

	if runErr == nil {
		t.Fatal("run extract PDF: expected error for Red_Hat PDF, got nil")
	}
	errStr := runErr.Error()
	if !containsString(errStr, "zlib") && !containsString(errStr, "invalid header") {
		t.Fatalf("error = %q, want contains 'zlib' or 'invalid header'", errStr)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
