package render

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
	"github.com/PolarKits/polar-doc/internal/ofd"
	"github.com/PolarKits/polar-doc/internal/pdf"
)

func TestNewPDFEngine(t *testing.T) {
	pdfSvc := pdf.NewService()
	engine := NewPDFEngine(pdfSvc)
	if engine == nil {
		t.Fatal("NewPDFEngine returned nil")
	}
}

func TestNewOFDEngine(t *testing.T) {
	ofdSvc := ofd.NewService()
	engine := NewOFDEngine(ofdSvc)
	if engine == nil {
		t.Fatal("NewOFDEngine returned nil")
	}
}

func TestFormatEngines_NewFormatEngines(t *testing.T) {
	fe := NewFormatEngines()
	if fe == nil {
		t.Fatal("NewFormatEngines returned nil")
	}
}

func TestFormatEngines_Register(t *testing.T) {
	fe := NewFormatEngines()
	pdfSvc := pdf.NewService()
	pdfEngine := NewPDFEngine(pdfSvc)

	err := fe.Register(doc.FormatPDF, pdfEngine)
	if err != nil {
		t.Fatalf("Register: unexpected error: %v", err)
	}
}

func TestFormatEngines_Register_Duplicate(t *testing.T) {
	fe := NewFormatEngines()
	pdfSvc := pdf.NewService()
	pdfEngine := NewPDFEngine(pdfSvc)

	err := fe.Register(doc.FormatPDF, pdfEngine)
	if err != nil {
		t.Fatalf("first Register: unexpected error: %v", err)
	}

	err = fe.Register(doc.FormatPDF, pdfEngine)
	if err == nil {
		t.Fatal("second Register: expected error, got nil")
	}
}

func TestFormatEngines_Engine(t *testing.T) {
	fe := NewFormatEngines()
	pdfSvc := pdf.NewService()
	pdfEngine := NewPDFEngine(pdfSvc)

	fe.Register(doc.FormatPDF, pdfEngine)

	engine, ok := fe.Engine(doc.FormatPDF)
	if !ok {
		t.Fatal("Engine: format not found")
	}
	if engine == nil {
		t.Fatal("Engine: returned nil")
	}
}

func TestFormatEngines_Engine_NotFound(t *testing.T) {
	fe := NewFormatEngines()

	_, ok := fe.Engine(doc.FormatPDF)
	if ok {
		t.Fatal("Engine: expected not found, got true")
	}
}

func TestFormatEngines_RenderPreview_NoEngine(t *testing.T) {
	fe := NewFormatEngines()
	_, err := fe.RenderPreview(context.Background(), &mockDoc{}, doc.PreviewRequest{})
	if err == nil {
		t.Fatal("RenderPreview: expected error for unregistered format")
	}
	if !strings.Contains(err.Error(), "no engine registered for format") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "no engine registered for format")
	}
}

type mockDoc struct{}

func (m *mockDoc) Ref() doc.DocumentRef {
	return doc.DocumentRef{Format: doc.FormatPDF, Path: "/nonexistent.pdf"}
}

func (m *mockDoc) Close() error {
	return nil
}

func TestFormatEngines_RenderPreview_PDFEngine(t *testing.T) {
	fe := NewFormatEngines()
	pdfSvc := pdf.NewService()
	pdfEngine := NewPDFEngine(pdfSvc)
	fe.Register(doc.FormatPDF, pdfEngine)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pdf")
	if err := os.WriteFile(path, []byte(minimalPDF), 0o644); err != nil {
		t.Fatalf("write PDF: %v", err)
	}

	docRef := doc.DocumentRef{Format: doc.FormatPDF, Path: path}
	opened, openErr := pdfSvc.Open(context.Background(), docRef)
	if openErr != nil {
		t.Fatalf("open: %v", openErr)
	}
	t.Cleanup(func() { _ = opened.Close() })

	_, err := fe.RenderPreview(context.Background(), opened, doc.PreviewRequest{Page: 1})
	if err == nil {
		t.Fatal("RenderPreview: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "preview is not implemented") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "preview is not implemented")
	}
}

func TestFormatEngines_RenderPreview_OFDEngine(t *testing.T) {
	fe := NewFormatEngines()
	ofdSvc := ofd.NewService()
	ofdEngine := NewOFDEngine(ofdSvc)
	fe.Register(doc.FormatOFD, ofdEngine)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.ofd")
	if err := os.WriteFile(path, buildOFDTestPackage(), 0o644); err != nil {
		t.Fatalf("write OFD: %v", err)
	}

	docRef := doc.DocumentRef{Format: doc.FormatOFD, Path: path}
	opened, openErr := ofdSvc.Open(context.Background(), docRef)
	if openErr != nil {
		t.Fatalf("open: %v", openErr)
	}
	t.Cleanup(func() { _ = opened.Close() })

	_, err := fe.RenderPreview(context.Background(), opened, doc.PreviewRequest{Page: 1})
	if err == nil {
		t.Fatal("RenderPreview: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "preview is not implemented") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "preview is not implemented")
	}
}

func TestRegisterPDFEngine(t *testing.T) {
	services := make(map[doc.Format]Engine)
	pdfSvc := pdf.NewService()

	RegisterPDFEngine(services, pdfSvc)

	if _, ok := services[doc.FormatPDF]; !ok {
		t.Fatal("RegisterPDFEngine: format not registered")
	}
}

func TestRegisterOFDEngine(t *testing.T) {
	services := make(map[doc.Format]Engine)
	ofdSvc := ofd.NewService()

	RegisterOFDEngine(services, ofdSvc)

	if _, ok := services[doc.FormatOFD]; !ok {
		t.Fatal("RegisterOFDEngine: format not registered")
	}
}

func TestRegisterPDFEngine_Duplicate(t *testing.T) {
	services := make(map[doc.Format]Engine)
	pdfSvc := pdf.NewService()

	RegisterPDFEngine(services, pdfSvc)
	RegisterPDFEngine(services, pdfSvc)

	if _, ok := services[doc.FormatPDF]; !ok {
		t.Fatal("RegisterPDFEngine: format not registered after duplicate")
	}
}

func buildOFDTestPackage() []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016" Version="1.0"><ofd:DocBody><ofd:DocRoot>Doc_0/Document.xml</ofd:DocRoot></ofd:DocBody></ofd:OFD>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:Pages><ofd:Page ID="0"/></ofd:Pages></ofd:Document>`,
	}
	for name, body := range files {
		w, _ := zw.Create(name)
		w.Write([]byte(body))
	}
	zw.Close()
	return buf.Bytes()
}

const minimalPDF = `%PDF-1.4
1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj
2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj
3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >> endobj
xref
0 4
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
trailer << /Size 4 /Root 1 0 R >>
startxref
190
%%EOF`