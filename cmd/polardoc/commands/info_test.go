package commands

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

func TestRunInfoPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.4\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{path}); err != nil {
			t.Fatalf("run info PDF: %v", err)
		}
	})

	mustContain(t, output, "format: pdf")
	mustContain(t, output, "path: "+path)
	mustContain(t, output, "size_bytes: 9")
	mustContain(t, output, "declared_version: 1.4")
}

func TestRunInfoOFD(t *testing.T) {
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
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{path}); err != nil {
			t.Fatalf("run info OFD: %v", err)
		}
	})

	mustContain(t, output, "format: ofd")
	mustContain(t, output, "path: "+path)
	mustContain(t, output, "size_bytes: ")
	if strings.Contains(output, "declared_version:") {
		t.Fatalf("unexpected declared_version output: %q", output)
	}
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}

	os.Stdout = w
	run()
	_ = w.Close()
	os.Stdout = oldStdout

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(data)
}

func mustContain(t *testing.T, output, expected string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Fatalf("output = %q, want contains %q", output, expected)
	}
}

func buildOFDPackage(t *testing.T, files map[string]string) []byte {
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
