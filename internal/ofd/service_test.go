package ofd

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarKits/polardoc/internal/doc"
	"github.com/PolarKits/polardoc/internal/pdf"
)

func TestServiceOpenAndInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: path})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	info, err := svc.Info(context.Background(), d)
	if err != nil {
		t.Fatalf("info OFD: %v", err)
	}

	if info.Format != doc.FormatOFD {
		t.Fatalf("format = %q, want %q", info.Format, doc.FormatOFD)
	}
	if info.Path != path {
		t.Fatalf("path = %q, want %q", info.Path, path)
	}
	if info.SizeBytes != int64(len(content)) {
		t.Fatalf("size = %d, want %d", info.SizeBytes, len(content))
	}
	if info.DeclaredVersion != "" {
		t.Fatalf("declared version = %q, want empty", info.DeclaredVersion)
	}
}

func TestServiceValidateValidOFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: path})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	report, err := svc.Validate(context.Background(), d)
	if err != nil {
		t.Fatalf("validate OFD: %v", err)
	}

	if !report.Valid {
		t.Fatalf("valid = false, want true")
	}
	if len(report.Errors) != 0 {
		t.Fatalf("errors = %v, want empty", report.Errors)
	}
}

func TestServiceValidateInvalidOFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml": "<ofd/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write bad OFD: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: path})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	report, err := svc.Validate(context.Background(), d)
	if err != nil {
		t.Fatalf("validate OFD: %v", err)
	}

	if report.Valid {
		t.Fatal("valid = true, want false")
	}
	if len(report.Errors) != 1 {
		t.Fatalf("errors = %v, want one error", report.Errors)
	}
	if report.Errors[0] != "invalid OFD package: missing Document.xml" {
		t.Fatalf("error = %q, want %q", report.Errors[0], "invalid OFD package: missing Document.xml")
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

func TestServiceExtractTextRejectsWrongDocumentType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	pdfSvc := pdf.NewService()
	pdfDoc, err := pdfSvc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF: %v", err)
	}
	t.Cleanup(func() { _ = pdfDoc.Close() })

	ofdSvc := NewService()
	_, err = ofdSvc.ExtractText(context.Background(), pdfDoc)
	if err == nil {
		t.Fatalf("ExtractText with PDF doc: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported document type") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "unsupported document type")
	}
}

func TestServiceRenderPreviewRejectsWrongDocumentType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	pdfSvc := pdf.NewService()
	pdfDoc, err := pdfSvc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF: %v", err)
	}
	t.Cleanup(func() { _ = pdfDoc.Close() })

	ofdSvc := NewService()
	_, err = ofdSvc.RenderPreview(context.Background(), pdfDoc, doc.PreviewRequest{})
	if err == nil {
		t.Fatalf("RenderPreview with PDF doc: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported document type") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "unsupported document type")
	}
}

func TestServiceExtractTextNotImplemented(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: path})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	_, err = svc.ExtractText(context.Background(), d)
	if err == nil {
		t.Fatal("ExtractText expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("error = %q, want contains 'not implemented'", err.Error())
	}
}
