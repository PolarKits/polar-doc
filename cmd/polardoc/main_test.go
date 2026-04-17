package main

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarKits/polardoc/internal/app"
)

func TestRunInfoSuccessReturnsZero(t *testing.T) {
	path := writeTestPDF(t)
	resolver := app.NewPhase1Resolver()

	stdout, stderr, code := captureProcessIO(t, func(errWriter io.Writer) int {
		return run(context.Background(), []string{"info", path}, resolver, errWriter)
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "format: pdf") {
		t.Fatalf("stdout = %q, want info output", stdout)
	}
}

func TestRunInfoFailureReturnsNonZeroAndUsesStderr(t *testing.T) {
	resolver := app.NewPhase1Resolver()

	stdout, stderr, code := captureProcessIO(t, func(errWriter io.Writer) int {
		return run(context.Background(), []string{"info", "/tmp/missing.pdf"}, resolver, errWriter)
	})

	if code == 0 {
		t.Fatalf("exit code = %d, want non-zero", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "error:") {
		t.Fatalf("stderr = %q, want error prefix", stderr)
	}
}

func TestRunValidateFailureReturnsNonZero(t *testing.T) {
	resolver := app.NewPhase1Resolver()

	stdout, stderr, code := captureProcessIO(t, func(errWriter io.Writer) int {
		return run(context.Background(), []string{"validate"}, resolver, errWriter)
	})

	if code == 0 {
		t.Fatalf("exit code = %d, want non-zero", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "error:") {
		t.Fatalf("stderr = %q, want error prefix", stderr)
	}
}

func TestRunUnknownCommandReturnsNonZero(t *testing.T) {
	resolver := app.NewPhase1Resolver()

	stdout, stderr, code := captureProcessIO(t, func(errWriter io.Writer) int {
		return run(context.Background(), []string{"unknown"}, resolver, errWriter)
	})

	if code == 0 {
		t.Fatalf("exit code = %d, want non-zero", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, `unknown command "unknown"`) {
		t.Fatalf("stderr = %q, want unknown command error", stderr)
	}
}

func TestRunValidateSuccessKeepsStdoutCleanForJSON(t *testing.T) {
	path := writeTestOFD(t)
	resolver := app.NewPhase1Resolver()

	stdout, stderr, code := captureProcessIO(t, func(errWriter io.Writer) int {
		return run(context.Background(), []string{"validate", "--json", path}, resolver, errWriter)
	})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, `"valid": true`) {
		t.Fatalf("stdout = %q, want validate JSON output", stdout)
	}
}

func captureProcessIO(t *testing.T, runFunc func(io.Writer) int) (string, string, int) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	code := runFunc(stderrWriter)

	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	stdoutData, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderrData, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}

	return string(stdoutData), string(stderrData), code
}

func writeTestPDF(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}
	return path
}

func writeTestOFD(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "sample.ofd")
	content := buildTestOFDPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}
	return path
}

func buildTestOFDPackage(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}
