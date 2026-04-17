package pdf

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarKits/polardoc/internal/doc"
)

func TestServiceOpenAndInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	info, err := svc.Info(context.Background(), d)
	if err != nil {
		t.Fatalf("info PDF: %v", err)
	}

	if info.Format != doc.FormatPDF {
		t.Fatalf("format = %q, want %q", info.Format, doc.FormatPDF)
	}
	if info.Path != path {
		t.Fatalf("path = %q, want %q", info.Path, path)
	}
	if info.SizeBytes != int64(len(content)) {
		t.Fatalf("size = %d, want %d", info.SizeBytes, len(content))
	}
	if info.DeclaredVersion != "1.7" {
		t.Fatalf("declared version = %q, want %q", info.DeclaredVersion, "1.7")
	}
}

func TestServiceValidateValidPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	report, err := svc.Validate(context.Background(), d)
	if err != nil {
		t.Fatalf("validate PDF: %v", err)
	}

	if !report.Valid {
		t.Fatalf("valid = false, want true")
	}
	if len(report.Errors) != 0 {
		t.Fatalf("errors = %v, want empty", report.Errors)
	}
}

func TestServiceValidateInvalidPDFHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pdf")
	if err := os.WriteFile(path, []byte("NOT_A_PDF\n"), 0o644); err != nil {
		t.Fatalf("write bad PDF: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	report, err := svc.Validate(context.Background(), d)
	if err != nil {
		t.Fatalf("validate PDF: %v", err)
	}

	if report.Valid {
		t.Fatal("valid = true, want false")
	}
	if len(report.Errors) != 1 {
		t.Fatalf("errors = %v, want one error", report.Errors)
	}
	if report.Errors[0] != "invalid PDF header" {
		t.Fatalf("error = %q, want %q", report.Errors[0], "invalid PDF header")
	}
}
