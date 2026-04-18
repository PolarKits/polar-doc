package pdf

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarKits/polardoc/internal/doc"
	"github.com/PolarKits/polardoc/internal/ofd"
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

func TestServiceOpenAndInfoPDF20(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-2.0\n1 0 obj\n<<>>\nendobj\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF 2.0: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF 2.0: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	info, err := svc.Info(context.Background(), d)
	if err != nil {
		t.Fatalf("info PDF 2.0: %v", err)
	}

	if info.DeclaredVersion != "2.0" {
		t.Fatalf("declared version = %q, want %q", info.DeclaredVersion, "2.0")
	}
}

func TestServiceValidateValidPDF20(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-2.0\n1 0 obj\n<<>>\nendobj\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF 2.0: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF 2.0: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	report, err := svc.Validate(context.Background(), d)
	if err != nil {
		t.Fatalf("validate PDF 2.0: %v", err)
	}

	if !report.Valid {
		t.Fatalf("valid = false, want true for PDF 2.0 header")
	}
}

func TestServiceOpenAndInfoPDF14(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF 1.4: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF 1.4: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	info, err := svc.Info(context.Background(), d)
	if err != nil {
		t.Fatalf("info PDF 1.4: %v", err)
	}

	if info.DeclaredVersion != "1.4" {
		t.Fatalf("declared version = %q, want %q", info.DeclaredVersion, "1.4")
	}
}

func TestServiceValidateValidPDF14(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.4\n1 0 obj\n<<>>\nendobj\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF 1.4: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF 1.4: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	report, err := svc.Validate(context.Background(), d)
	if err != nil {
		t.Fatalf("validate PDF 1.4: %v", err)
	}

	if !report.Valid {
		t.Fatalf("valid = false, want true for PDF 1.4 header")
	}
}

func TestServiceOpenAndInfoPDF13PreRelease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.3\n1 0 obj\n<<>>\nendobj\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF 1.3: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF 1.3: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	info, err := svc.Info(context.Background(), d)
	if err != nil {
		t.Fatalf("info PDF 1.3: %v", err)
	}

	if info.DeclaredVersion != "1.3" {
		t.Fatalf("declared version = %q, want %q", info.DeclaredVersion, "1.3")
	}
}

func TestServiceValidateValidPDF13PreRelease(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	if err := os.WriteFile(path, []byte("%PDF-1.3\n1 0 obj\n<<>>\nendobj\n"), 0o644); err != nil {
		t.Fatalf("write sample PDF 1.3: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF 1.3: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	report, err := svc.Validate(context.Background(), d)
	if err != nil {
		t.Fatalf("validate PDF 1.3: %v", err)
	}

	if !report.Valid {
		t.Fatalf("valid = false, want true for PDF 1.3 header")
	}
}

func TestServiceReadStartxrefFindsXrefOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [] /Count 0 >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 3\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 3 >>\n" +
		"startxref\n" +
		"110\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	offset, err := readStartxref(f)
	if err != nil {
		t.Fatalf("readStartxref: %v", err)
	}
	if offset != 110 {
		t.Fatalf("startxref offset = %d, want 110", offset)
	}
}

func TestServiceReadTrailerReturnsRootRef(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [] /Count 0 >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 3\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 3 >>\n" +
		"startxref\n" +
		"110\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	rootRef, err := readTrailerRootRef(f, 110)
	if err != nil {
		t.Fatalf("readTrailerRootRef: %v", err)
	}
	if rootRef != "1 0 R" {
		t.Fatalf("root ref = %q, want %q", rootRef, "1 0 R")
	}
}

func TestServiceReadCatalogObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [] /Count 0 >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 3\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 3 >>\n" +
		"startxref\n" +
		"110\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	catalog, err := readObject(f, "1 0 R")
	if err != nil {
		t.Fatalf("readObject(catalog): %v", err)
	}
	if !strings.Contains(catalog, "/Type /Catalog") {
		t.Fatalf("catalog = %q, want contains /Type /Catalog", catalog)
	}
	if !strings.Contains(catalog, "/Pages 2 0 R") {
		t.Fatalf("catalog = %q, want contains /Pages 2 0 R", catalog)
	}
}

func TestServiceReadStartxrefMissingStartxref(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [] /Count 0 >>\nendobj\n" +
		"xref\n" +
		"0 3\n" +
		"0000000000 65535 f \n" +
		"0000000069 00000 n \n" +
		"0000000138 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 3 >>\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = readStartxref(f)
	if err == nil {
		t.Fatal("readStartxref: expected error for missing startxref, got nil")
	}
}

func TestServiceReadTrailerMissingRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [] /Count 0 >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 3\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"trailer\n" +
		"<< /Size 3 >>\n" +
		"startxref\n" +
		"203\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = readTrailerRootRef(f, 203)
	if err == nil {
		t.Fatal("readTrailerRootRef: expected error for missing Root, got nil")
	}
}

func TestServiceExtractTextRejectsWrongDocumentType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDTestPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	ofdSvc := ofd.NewService()
	ofdDoc, err := ofdSvc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: path})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = ofdDoc.Close() })

	pdfSvc := NewService()
	_, err = pdfSvc.ExtractText(context.Background(), ofdDoc)
	if err == nil {
		t.Fatalf("ExtractText with OFD doc: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported document type") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "unsupported document type")
	}
}

func TestServiceExtractTextReturnsEmpty(t *testing.T) {
	svc := NewService()
	samples := []struct {
		name string
		path string
	}{
		{"pdf20-utf8", filepath.Join("..", "..", "testdata", "pdf", "pdf20-utf8-test.pdf")},
		{"redhat-openshift", filepath.Join("..", "..", "testdata", "pdf", "Red_Hat_OpenShift_Serverless-1.35-Serverless_Logic-en-US.pdf")},
		{"sample-local-pdf", filepath.Join("..", "..", "testdata", "pdf", "sample-local-pdf.pdf")},
		{"testPDF-5x", filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.5.x.pdf")},
		{"testPDF-8x", filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.8.x.pdf")},
	}

	for _, tc := range samples {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := os.Stat(tc.path); os.IsNotExist(err) {
				t.Skipf("%s not found", tc.name)
			}

			d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: tc.path})
			if err != nil {
				t.Fatalf("open PDF: %v", err)
			}
			defer d.Close()

			result, err := svc.ExtractText(context.Background(), d)
			if err != nil {
				t.Fatalf("ExtractText should not error, got: %v", err)
			}
			if result.Text != "" {
				t.Fatalf("ExtractText returns non-empty text %q, want empty string (not implemented)", result.Text)
			}
		})
	}
}

func TestServiceReadTrailerRootAtObject3(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Title (Test) >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [] /Count 0 >>\n" +
		"endobj\n" +
		"3 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 4\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000044 00000 n \n" +
		"0000000096 00000 n \n" +
		"trailer\n" +
		"<< /Root 3 0 R /Size 4 >>\n" +
		"startxref\n" +
		"145\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	rootRef, err := readTrailerRootRef(f, 145)
	if err != nil {
		t.Fatalf("readTrailerRootRef: %v", err)
	}
	if rootRef != "3 0 R" {
		t.Fatalf("root ref = %q, want %q", rootRef, "3 0 R")
	}

	catalog, err := readObject(f, rootRef)
	if err != nil {
		t.Fatalf("readObject(catalog): %v", err)
	}
	if !strings.Contains(catalog, "/Type /Catalog") {
		t.Fatalf("catalog = %q, want contains /Type /Catalog", catalog)
	}
}

func TestServiceReadObjectWithNonZeroGeneration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 1 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [] /Count 0 >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 3\n" +
		"0000000000 65535 f \n" +
		"0000000009 00001 n \n" +
		"0000000058 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 1 R /Size 3 >>\n" +
		"startxref\n" +
		"110\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	rootRef := "1 1 R"
	catalog, err := readObject(f, rootRef)
	if err != nil {
		t.Fatalf("readObject(gen1): %v", err)
	}
	if !strings.Contains(catalog, "/Type /Catalog") {
		t.Fatalf("catalog = %q, want contains /Type /Catalog", catalog)
	}
}

func TestServiceRenderPreviewRejectsWrongDocumentType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDTestPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	ofdSvc := ofd.NewService()
	ofdDoc, err := ofdSvc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: path})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = ofdDoc.Close() })

	pdfSvc := NewService()
	_, err = pdfSvc.RenderPreview(context.Background(), ofdDoc, doc.PreviewRequest{})
	if err == nil {
		t.Fatalf("RenderPreview with OFD doc: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported document type") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "unsupported document type")
	}
}

func TestServiceReadPagesFromCatalog(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n" +
		"endobj\n" +
		"3 0 obj\n" +
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 4\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 4 >>\n" +
		"startxref\n" +
		"186\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	catalogStr, err := readObject(f, "1 0 R")
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}

	pagesRef, err := readPagesRefFromCatalog(catalogStr)
	if err != nil {
		t.Fatalf("readPagesRefFromCatalog: %v", err)
	}
	if pagesRef != "2 0 R" {
		t.Fatalf("pages ref = %q, want %q", pagesRef, "2 0 R")
	}

	pagesObj, err := readObject(f, pagesRef)
	if err != nil {
		t.Fatalf("read pages object: %v", err)
	}

	kids, count, err := readPagesKids(pagesObj)
	if err != nil {
		t.Fatalf("readPagesKids: %v", err)
	}
	if len(kids) != 1 || kids[0] != "3 0 R" {
		t.Fatalf("kids = %v, want %v", kids, []string{"3 0 R"})
	}
	if count != 1 {
		t.Fatalf("count = %d, want %d", count, 1)
	}

	pageObj, err := readObject(f, kids[0])
	if err != nil {
		t.Fatalf("read page object: %v", err)
	}
	if !strings.Contains(pageObj, "/Type /Page") {
		t.Fatalf("page = %q, want contains /Type /Page", pageObj)
	}
	if !strings.Contains(pageObj, "/MediaBox") {
		t.Fatalf("page = %q, want contains /MediaBox", pageObj)
	}
}

func TestServiceReadPagesMissingKids(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Count 0 >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 3\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 3 >>\n" +
		"startxref\n" +
		"101\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	catalogStr, err := readObject(f, "1 0 R")
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}

	pagesRef, err := readPagesRefFromCatalog(catalogStr)
	if err != nil {
		t.Fatalf("readPagesRefFromCatalog: %v", err)
	}

	pagesObj, err := readObject(f, pagesRef)
	if err != nil {
		t.Fatalf("read pages object: %v", err)
	}

	_, _, err = readPagesKids(pagesObj)
	if err == nil {
		t.Fatal("readPagesKids: expected error for missing /Kids, got nil")
	}
}

func TestServiceReadPagesKidsCountMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 1 >>\n" +
		"endobj\n" +
		"3 0 obj\n" +
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\n" +
		"endobj\n" +
		"4 0 obj\n" +
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 5\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000186 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 5 >>\n" +
		"startxref\n" +
		"263\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	catalogStr, err := readObject(f, "1 0 R")
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}

	pagesRef, err := readPagesRefFromCatalog(catalogStr)
	if err != nil {
		t.Fatalf("readPagesRefFromCatalog: %v", err)
	}

	pagesObj, err := readObject(f, pagesRef)
	if err != nil {
		t.Fatalf("read pages object: %v", err)
	}

	_, _, err = readPagesKids(pagesObj)
	if err == nil {
		t.Fatalf("readPagesKids: expected /Count mismatch error, got nil")
	}
}

func TestServiceReadPageFromKidsNonPageObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n" +
		"endobj\n" +
		"3 0 obj\n" +
		"<< /Type /Action /S /JavaScript /JS (alert) >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 4\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 4 >>\n" +
		"startxref\n" +
		"165\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	catalogStr, err := readObject(f, "1 0 R")
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}

	pagesRef, err := readPagesRefFromCatalog(catalogStr)
	if err != nil {
		t.Fatalf("readPagesRefFromCatalog: %v", err)
	}

	pagesObj, err := readObject(f, pagesRef)
	if err != nil {
		t.Fatalf("read pages object: %v", err)
	}

	kids, _, err := readPagesKids(pagesObj)
	if err != nil {
		t.Fatalf("readPagesKids: %v", err)
	}

	_, err = readPageFromKids(f, kids[0])
	if err == nil {
		t.Fatalf("readPageFromKids: expected error for /Type /Action kid, got nil")
	}
}

func buildOFDTestPackage(t *testing.T, files map[string]string) []byte {
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

func TestServiceReadNestedPagesTree(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n" +
		"endobj\n" +
		"3 0 obj\n" +
		"<< /Type /Pages /Kids [4 0 R] /Count 1 >>\n" +
		"endobj\n" +
		"4 0 obj\n" +
		"<< /Type /Page /Parent 3 0 R /MediaBox [0 0 612 792] >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 5\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000172 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 5 >>\n" +
		"startxref\n" +
		"243\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	catalogStr, err := readObject(f, "1 0 R")
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}

	pagesRef, err := readPagesRefFromCatalog(catalogStr)
	if err != nil {
		t.Fatalf("readPagesRefFromCatalog: %v", err)
	}
	if pagesRef != "2 0 R" {
		t.Fatalf("root pages ref = %q, want %q", pagesRef, "2 0 R")
	}

	rootPagesObj, err := readObject(f, pagesRef)
	if err != nil {
		t.Fatalf("read root pages object: %v", err)
	}

	rootKids, _, err := readPagesKids(rootPagesObj)
	if err != nil {
		t.Fatalf("readPagesKids: %v", err)
	}
	if len(rootKids) != 1 || rootKids[0] != "3 0 R" {
		t.Fatalf("root kids = %v, want %v", rootKids, []string{"3 0 R"})
	}

	intermediatePagesObj, err := readObject(f, rootKids[0])
	if err != nil {
		t.Fatalf("read intermediate pages object: %v", err)
	}

	intermediateKids, _, err := readPagesKids(intermediatePagesObj)
	if err != nil {
		t.Fatalf("readPagesKids: %v", err)
	}
	if len(intermediateKids) != 1 || intermediateKids[0] != "4 0 R" {
		t.Fatalf("intermediate kids = %v, want %v", intermediateKids, []string{"4 0 R"})
	}

	pageObj, err := readPageFromKids(f, intermediateKids[0])
	if err != nil {
		t.Fatalf("readPageFromKids: %v", err)
	}
	if !strings.Contains(pageObj, "/Type /Page") {
		t.Fatalf("page obj = %q, want /Type /Page", pageObj)
	}
	if !strings.Contains(pageObj, "/MediaBox") {
		t.Fatalf("page obj = %q, want /MediaBox", pageObj)
	}
}

func TestServiceReadPagesKidsWithNonPageNonPagesObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>\n" +
		"endobj\n" +
		"3 0 obj\n" +
		"<< /Type /Action /S /JavaScript >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 4\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 4 >>\n" +
		"startxref\n" +
		"165\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	catalogStr, err := readObject(f, "1 0 R")
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}

	pagesRef, err := readPagesRefFromCatalog(catalogStr)
	if err != nil {
		t.Fatalf("readPagesRefFromCatalog: %v", err)
	}

	pagesObj, err := readObject(f, pagesRef)
	if err != nil {
		t.Fatalf("read pages object: %v", err)
	}

	kids, _, err := readPagesKids(pagesObj)
	if err != nil {
		t.Fatalf("readPagesKids: %v", err)
	}

	_, err = readPageFromKids(f, kids[0])
	if err == nil {
		t.Fatalf("readPageFromKids: expected error for /Type /Action kid, got nil")
	}
}

func TestServiceReadPagesNodeMissingKids(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Count 0 >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 3\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 3 >>\n" +
		"startxref\n" +
		"101\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	catalogStr, err := readObject(f, "1 0 R")
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}

	pagesRef, err := readPagesRefFromCatalog(catalogStr)
	if err != nil {
		t.Fatalf("readPagesRefFromCatalog: %v", err)
	}

	pagesObj, err := readObject(f, pagesRef)
	if err != nil {
		t.Fatalf("read pages object: %v", err)
	}

	_, _, err = readPagesKids(pagesObj)
	if err == nil {
		t.Fatalf("readPagesKids: expected error for missing /Kids, got nil")
	}
}

func TestServiceReadPageResourcesAndContentsSingleRef(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000219 00000 n \n" +
		"0000000262 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"293\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	pageObj, err := readFirstPageFromPages(f, "2 0 R")
	if err != nil {
		t.Fatalf("readFirstPageFromPages: %v", err)
	}

	pageDict, err := extractDictFromObject(pageObj)
	if err != nil {
		t.Fatalf("extractDictFromObject: %v", err)
	}

	resRef, err := readPageResourcesRef(pageDict)
	if err != nil {
		t.Fatalf("readPageResourcesRef: %v", err)
	}
	if resRef.ObjNum != 4 || resRef.GenNum != 0 {
		t.Fatalf("resources ref = %v, want 4 0 R", RefToString(resRef))
	}

	contentsRefs, err := readPageContentsRefs(pageDict)
	if err != nil {
		t.Fatalf("readPageContentsRefs: %v", err)
	}
	if len(contentsRefs) != 1 {
		t.Fatalf("contents refs count = %d, want 1", len(contentsRefs))
	}
	if contentsRefs[0].ObjNum != 5 || contentsRefs[0].GenNum != 0 {
		t.Fatalf("contents ref = %v, want 5 0 R", RefToString(contentsRefs[0]))
	}
}

func TestServiceReadPageContentsSingleRef(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n" +
		"4 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 5\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000202 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 5 >>\n" +
		"startxref\n" +
		"233\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	pageObj, err := readFirstPageFromPages(f, "2 0 R")
	if err != nil {
		t.Fatalf("readFirstPageFromPages: %v", err)
	}

	pageDict, err := extractDictFromObject(pageObj)
	if err != nil {
		t.Fatalf("extractDictFromObject: %v", err)
	}

	contentsRefs, err := readPageContentsRefs(pageDict)
	if err != nil {
		t.Fatalf("readPageContentsRefs: %v", err)
	}
	if len(contentsRefs) != 1 {
		t.Fatalf("contents refs count = %d, want 1", len(contentsRefs))
	}
	if contentsRefs[0].ObjNum != 4 || contentsRefs[0].GenNum != 0 {
		t.Fatalf("contents ref = %v, want 4 0 R", RefToString(contentsRefs[0]))
	}
}

func TestServiceReadPageContentsRefArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents [4 0 R 5 0 R] >>\nendobj\n" +
		"4 0 obj\n<< /Length 0 >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000210 00000 n \n" +
		"0000000241 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"272\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	pageObj, err := readFirstPageFromPages(f, "2 0 R")
	if err != nil {
		t.Fatalf("readFirstPageFromPages: %v", err)
	}

	pageDict, err := extractDictFromObject(pageObj)
	if err != nil {
		t.Fatalf("extractDictFromObject: %v", err)
	}

	contentsRefs, err := readPageContentsRefs(pageDict)
	if err != nil {
		t.Fatalf("readPageContentsRefs: %v", err)
	}
	if len(contentsRefs) != 2 {
		t.Fatalf("contents refs count = %d, want 2", len(contentsRefs))
	}
	if contentsRefs[0].ObjNum != 4 || contentsRefs[0].GenNum != 0 {
		t.Fatalf("contents[0] = %v, want 4 0 R", RefToString(contentsRefs[0]))
	}
	if contentsRefs[1].ObjNum != 5 || contentsRefs[1].GenNum != 0 {
		t.Fatalf("contents[1] = %v, want 5 0 R", RefToString(contentsRefs[1]))
	}
}

func TestServiceReadPageMissingResources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n" +
		"4 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 5\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000202 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 5 >>\n" +
		"startxref\n" +
		"233\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	pageObj, err := readFirstPageFromPages(f, "2 0 R")
	if err != nil {
		t.Fatalf("readFirstPageFromPages: %v", err)
	}

	pageDict, err := extractDictFromObject(pageObj)
	if err != nil {
		t.Fatalf("extractDictFromObject: %v", err)
	}

	_, err = readPageResourcesRef(pageDict)
	if err == nil {
		t.Fatal("readPageResourcesRef: expected error for missing /Resources, got nil")
	}
}

func TestServiceReadPageMissingContents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"xref\n" +
		"0 5\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000203 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 5 >>\n" +
		"startxref\n" +
		"246\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	pageObj, err := readFirstPageFromPages(f, "2 0 R")
	if err != nil {
		t.Fatalf("readFirstPageFromPages: %v", err)
	}

	pageDict, err := extractDictFromObject(pageObj)
	if err != nil {
		t.Fatalf("extractDictFromObject: %v", err)
	}

	_, err = readPageContentsRefs(pageDict)
	if err == nil {
		t.Fatal("readPageContentsRefs: expected error for missing /Contents, got nil")
	}
}

func TestServiceReadPageContentsNotRefOrArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents /Direct >>\nendobj\n" +
		"xref\n" +
		"0 4\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 4 >>\n" +
		"startxref\n" +
		"204\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	pageObj, err := readFirstPageFromPages(f, "2 0 R")
	if err != nil {
		t.Fatalf("readFirstPageFromPages: %v", err)
	}

	pageDict, err := extractDictFromObject(pageObj)
	if err != nil {
		t.Fatalf("extractDictFromObject: %v", err)
	}

	_, err = readPageContentsRefs(pageDict)
	if err == nil {
		t.Fatal("readPageContentsRefs: expected error for /Contents that is not ref or array, got nil")
	}
}

func TestReadFirstPageInfoContentsSingleRef(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000219 00000 n \n" +
		"0000000262 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"293\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.PagesRef.ObjNum != 2 || info.PagesRef.GenNum != 0 {
		t.Fatalf("PagesRef = %v, want 2 0 R", RefToString(info.PagesRef))
	}
	if info.PageRef.ObjNum != 3 || info.PageRef.GenNum != 0 {
		t.Fatalf("PageRef = %v, want 3 0 R", RefToString(info.PageRef))
	}
	if info.Parent.ObjNum != 2 || info.Parent.GenNum != 0 {
		t.Fatalf("Parent = %v, want 2 0 R", RefToString(info.Parent))
	}
	if len(info.MediaBox) != 4 {
		t.Fatalf("MediaBox length = %d, want 4", len(info.MediaBox))
	}
	if info.Resources.ObjNum != 4 || info.Resources.GenNum != 0 {
		t.Fatalf("Resources = %v, want 4 0 R", RefToString(info.Resources))
	}
	if len(info.Contents) != 1 {
		t.Fatalf("Contents length = %d, want 1 (single ref)", len(info.Contents))
	}
	if info.Contents[0].ObjNum != 5 || info.Contents[0].GenNum != 0 {
		t.Fatalf("Contents[0] = %v, want 5 0 R", RefToString(info.Contents[0]))
	}
}

func TestReadFirstPageInfoContentsArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R /Contents [5 0 R 6 0 R] >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"6 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 7\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000225 00000 n \n" +
		"0000000268 00000 n \n" +
		"0000000311 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 7 >>\n" +
		"startxref\n" +
		"332\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if len(info.Contents) != 2 {
		t.Fatalf("Contents length = %d, want 2 (array)", len(info.Contents))
	}
	if info.Contents[0].ObjNum != 5 || info.Contents[0].GenNum != 0 {
		t.Fatalf("Contents[0] = %v, want 5 0 R", RefToString(info.Contents[0]))
	}
	if info.Contents[1].ObjNum != 6 || info.Contents[1].GenNum != 0 {
		t.Fatalf("Contents[1] = %v, want 6 0 R", RefToString(info.Contents[1]))
	}
}

func TestReadFirstPageInfoTrailerMissingRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000219 00000 n \n" +
		"0000000262 00000 n \n" +
		"trailer\n" +
		"<< /Size 6 >>\n" +
		"startxref\n" +
		"277\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo: expected error for missing /Root in trailer, got nil")
	}
}

func TestReadFirstPageInfoNestedPagesTree(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Pages /Kids [4 0 R] /Count 1 >>\nendobj\n" +
		"4 0 obj\n<< /Type /Page /Parent 3 0 R /MediaBox [0 0 612 792] /Resources 5 0 R /Contents 6 0 R >>\nendobj\n" +
		"5 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"6 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 7\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000172 00000 n \n" +
		"0000000229 00000 n \n" +
		"0000000286 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 7 >>\n" +
		"startxref\n" +
		"335\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.PagesRef.ObjNum != 3 || info.PagesRef.GenNum != 0 {
		t.Fatalf("PagesRef = %v, want 3 0 R (immediate parent Pages)", RefToString(info.PagesRef))
	}
	if info.PageRef.ObjNum != 4 || info.PageRef.GenNum != 0 {
		t.Fatalf("PageRef = %v, want 4 0 R", RefToString(info.PageRef))
	}
	if info.Parent.ObjNum != 3 || info.Parent.GenNum != 0 {
		t.Fatalf("Parent = %v, want 3 0 R", RefToString(info.Parent))
	}
	if len(info.Contents) != 1 {
		t.Fatalf("Contents length = %d, want 1", len(info.Contents))
	}
}

func TestReadFirstPageInfoMissingResources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n" +
		"4 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 5\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000202 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 5 >>\n" +
		"startxref\n" +
		"233\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo: expected error for missing /Resources, got nil")
	}
	if !strings.Contains(err.Error(), "/Resources") {
		t.Fatalf("error = %q, want contains /Resources", err.Error())
	}
}

func TestReadFirstPageInfoMissingContents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"xref\n" +
		"0 5\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000203 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 5 >>\n" +
		"startxref\n" +
		"246\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo: expected error for missing /Contents, got nil")
	}
	if !strings.Contains(err.Error(), "/Contents") {
		t.Fatalf("error = %q, want contains /Contents", err.Error())
	}
}

func TestReadFirstPageInfoMissingMediaBox(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /Resources 4 0 R /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000195 00000 n \n" +
		"0000000238 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"269\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo: expected error for missing /MediaBox, got nil")
	}
	if !strings.Contains(err.Error(), "/MediaBox") {
		t.Fatalf("error = %q, want contains /MediaBox", err.Error())
	}
}

func TestReadFirstPageInfoMissingParent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /MediaBox [0 0 612 792] /Resources 4 0 R /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000205 00000 n \n" +
		"0000000248 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"279\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo: expected error for missing /Parent, got nil")
	}
	if !strings.Contains(err.Error(), "/Parent") {
		t.Fatalf("error = %q, want contains /Parent", err.Error())
	}
}

func TestReadFirstPageInfoMediaBoxFromParentPages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 612 792] >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /Resources 4 0 R /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000139 00000 n \n" +
		"0000000219 00000 n \n" +
		"0000000262 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"293\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.PageRef.ObjNum != 3 || info.PageRef.GenNum != 0 {
		t.Fatalf("PageRef = %v, want 3 0 R", RefToString(info.PageRef))
	}
	if len(info.MediaBox) != 4 {
		t.Fatalf("MediaBox length = %d, want 4", len(info.MediaBox))
	}
	mb0, ok0 := info.MediaBox[0].(PDFInteger)
	mb1, ok1 := info.MediaBox[1].(PDFInteger)
	mb2, ok2 := info.MediaBox[2].(PDFInteger)
	mb3, ok3 := info.MediaBox[3].(PDFInteger)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		t.Fatalf("MediaBox elements not PDFInteger: %v", info.MediaBox)
	}
	if int64(mb0) != 0 || int64(mb1) != 0 || int64(mb2) != 612 || int64(mb3) != 792 {
		t.Fatalf("MediaBox = %v, want [0 0 612 792]", info.MediaBox)
	}
}

func TestReadFirstPageInfoMediaBoxFromAncestorPages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Pages /Kids [4 0 R] /Count 1 /MediaBox [0 0 612 792] >>\nendobj\n" +
		"4 0 obj\n<< /Type /Page /Parent 3 0 R /Resources 5 0 R /Contents 6 0 R >>\nendobj\n" +
		"5 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"6 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 7\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000196 00000 n \n" +
		"0000000276 00000 n \n" +
		"0000000319 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 7 >>\n" +
		"startxref\n" +
		"350\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.PageRef.ObjNum != 4 || info.PageRef.GenNum != 0 {
		t.Fatalf("PageRef = %v, want 4 0 R", RefToString(info.PageRef))
	}
	if info.PagesRef.ObjNum != 3 || info.PagesRef.GenNum != 0 {
		t.Fatalf("PagesRef = %v, want 3 0 R (immediate parent)", RefToString(info.PagesRef))
	}
	if len(info.MediaBox) != 4 {
		t.Fatalf("MediaBox length = %d, want 4", len(info.MediaBox))
	}
	mb0, ok0 := info.MediaBox[0].(PDFInteger)
	mb1, ok1 := info.MediaBox[1].(PDFInteger)
	mb2, ok2 := info.MediaBox[2].(PDFInteger)
	mb3, ok3 := info.MediaBox[3].(PDFInteger)
	if !ok0 || !ok1 || !ok2 || !ok3 {
		t.Fatalf("MediaBox elements not PDFInteger: %v", info.MediaBox)
	}
	if int64(mb0) != 0 || int64(mb1) != 0 || int64(mb2) != 612 || int64(mb3) != 792 {
		t.Fatalf("MediaBox = %v, want [0 0 612 792] (inherited from ancestor)", info.MediaBox)
	}
}

func TestReadFirstPageInfoMediaBoxMissingInAllAncestors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /Resources 4 0 R /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000115 00000 n \n" +
		"0000000195 00000 n \n" +
		"0000000238 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"269\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo: expected error for /MediaBox missing in Page and all ancestors, got nil")
	}
	if !strings.Contains(err.Error(), "/MediaBox") {
		t.Fatalf("error = %q, want contains /MediaBox", err.Error())
	}
}

func TestReadFirstPageInfoResourcesFromParentPages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 612 792] /Resources 4 0 R >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000156 00000 n \n" +
		"0000000243 00000 n \n" +
		"0000000286 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"317\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.PageRef.ObjNum != 3 || info.PageRef.GenNum != 0 {
		t.Fatalf("PageRef = %v, want 3 0 R", RefToString(info.PageRef))
	}
	if info.Resources.ObjNum != 4 || info.Resources.GenNum != 0 {
		t.Fatalf("Resources = %v, want 4 0 R (inherited from parent Pages)", RefToString(info.Resources))
	}
}

func TestReadFirstPageInfoResourcesFromAncestorPages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 612 792] >>\nendobj\n" +
		"3 0 obj\n<< /Type /Pages /Kids [4 0 R] /Count 1 /MediaBox [0 0 612 792] /Resources 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /Type /Page /Parent 3 0 R /MediaBox [0 0 612 792] /Contents 6 0 R >>\nendobj\n" +
		"5 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"6 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 7\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000139 00000 n \n" +
		"0000000237 00000 n \n" +
		"0000000324 00000 n \n" +
		"0000000367 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 7 >>\n" +
		"startxref\n" +
		"398\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.PageRef.ObjNum != 4 || info.PageRef.GenNum != 0 {
		t.Fatalf("PageRef = %v, want 4 0 R", RefToString(info.PageRef))
	}
	if info.PagesRef.ObjNum != 3 || info.PagesRef.GenNum != 0 {
		t.Fatalf("PagesRef = %v, want 3 0 R (immediate parent)", RefToString(info.PagesRef))
	}
	if info.Resources.ObjNum != 5 || info.Resources.GenNum != 0 {
		t.Fatalf("Resources = %v, want 5 0 R (inherited from ancestor Pages)", RefToString(info.Resources))
	}
}

func TestReadFirstPageInfoResourcesMissingInAllAncestors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 612 792] >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>\nendobj\n" +
		"4 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 5\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000139 00000 n \n" +
		"0000000226 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 5 >>\n" +
		"startxref\n" +
		"257\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo: expected error for /Resources missing in Page and all ancestors, got nil")
	}
	if !strings.Contains(err.Error(), "/Resources") {
		t.Fatalf("error = %q, want contains /Resources", err.Error())
	}
}

func TestReadFirstPageInfoRotateInPage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 612 792] /Resources 4 0 R >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R /Contents 5 0 R /Rotate 90 >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000156 00000 n \n" +
		"0000000271 00000 n \n" +
		"0000000314 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"345\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.Rotate == nil {
		t.Fatal("Rotate = nil, want 90")
	}
	if *info.Rotate != 90 {
		t.Fatalf("Rotate = %d, want 90", *info.Rotate)
	}
}

func TestReadFirstPageInfoRotateFromParentPages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 612 792] /Resources 4 0 R /Rotate 180 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R /Contents 5 0 R >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000168 00000 n \n" +
		"0000000272 00000 n \n" +
		"0000000315 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"346\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.Rotate == nil {
		t.Fatal("Rotate = nil, want 180 (inherited from parent Pages)")
	}
	if *info.Rotate != 180 {
		t.Fatalf("Rotate = %d, want 180", *info.Rotate)
	}
}

func TestReadFirstPageInfoRotateFromAncestorPages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 612 792] /Resources 4 0 R /Rotate 270 >>\nendobj\n" +
		"3 0 obj\n<< /Type /Pages /Kids [4 0 R] /Count 1 /MediaBox [0 0 612 792] /Resources 5 0 R /Parent 2 0 R >>\nendobj\n" +
		"4 0 obj\n<< /Type /Page /Parent 3 0 R /MediaBox [0 0 612 792] /Resources 5 0 R /Contents 6 0 R >>\nendobj\n" +
		"5 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"6 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 7\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000168 00000 n \n" +
		"0000000280 00000 n \n" +
		"0000000384 00000 n \n" +
		"0000000427 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 7 >>\n" +
		"startxref\n" +
		"458\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.Rotate == nil {
		t.Fatal("Rotate = nil, want 270 (inherited from ancestor Pages)")
	}
	if *info.Rotate != 270 {
		t.Fatalf("Rotate = %d, want 270", *info.Rotate)
	}
}

func TestReadFirstPageInfoRotateNotInteger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 612 792] /Resources 4 0 R >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R /Contents 5 0 R /Rotate /Invalid >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000156 00000 n \n" +
		"0000000277 00000 n \n" +
		"0000000320 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"351\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo: expected error for /Rotate that is not an integer, got nil")
	}
	if !strings.Contains(err.Error(), "/Rotate") {
		t.Fatalf("error = %q, want contains /Rotate", err.Error())
	}
}

func TestReadFirstPageInfoRotateZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n" +
		"2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 /MediaBox [0 0 612 792] /Resources 4 0 R >>\nendobj\n" +
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources 4 0 R /Contents 5 0 R /Rotate 0 >>\nendobj\n" +
		"4 0 obj\n<< /ProcSet [/PDF /Text] >>\nendobj\n" +
		"5 0 obj\n<< /Length 0 >>\nendobj\n" +
		"xref\n" +
		"0 6\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000156 00000 n \n" +
		"0000000270 00000 n \n" +
		"0000000313 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 6 >>\n" +
		"startxref\n" +
		"344\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	info, err := ReadFirstPageInfo(f)
	if err != nil {
		t.Fatalf("ReadFirstPageInfo: %v", err)
	}

	if info.Rotate == nil {
		t.Fatal("Rotate = nil, but Rotate=0 should be distinguishable from absent")
	}
	if *info.Rotate != 0 {
		t.Fatalf("Rotate = %d, want 0", *info.Rotate)
	}
}
