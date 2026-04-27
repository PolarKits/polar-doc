package ofd

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PolarKits/polar-doc/internal/doc"
	"github.com/PolarKits/polar-doc/internal/pdf"
)

// TestFirstPageInfo_WithPhysicalBox verifies that FirstPageInfo returns
// the PhysicalBox mapped to MediaBox when present in Document.xml.
func TestFirstPageInfo_WithPhysicalBox(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016" Version="1.0"><ofd:DocBody><ofd:DocRoot>Doc_0/Document.xml</ofd:DocRoot></ofd:DocBody></ofd:OFD>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:CommonData><ofd:PageArea><ofd:PhysicalBox>0 0 210 297</ofd:PhysicalBox></ofd:PageArea></ofd:CommonData><ofd:Pages><ofd:Page ID="1"/></ofd:Pages></ofd:Document>`,
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

	result, err := svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		t.Fatalf("FirstPageInfo: unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("FirstPageInfo: expected non-nil result")
	}
	if result.Path != path {
		t.Fatalf("Path = %q, want %q", result.Path, path)
	}
	if len(result.MediaBox) != 4 {
		t.Fatalf("MediaBox len = %d, want 4", len(result.MediaBox))
	}
	// PhysicalBox "0 0 210 297" (x y w h) → MediaBox [0, 0, 210, 297] (llx lly urx ury)
	if result.MediaBox[0] != 0 || result.MediaBox[1] != 0 || result.MediaBox[2] != 210 || result.MediaBox[3] != 297 {
		t.Fatalf("MediaBox = %v, want [0, 0, 210, 297]", result.MediaBox)
	}
}

// TestFirstPageInfo_NoPhysicalBox verifies that FirstPageInfo returns
// (nil, nil) when PhysicalBox is absent, not an error.
func TestFirstPageInfo_NoPhysicalBox(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016" Version="1.0"><ofd:DocBody><ofd:DocRoot>Doc_0/Document.xml</ofd:DocRoot></ofd:DocBody></ofd:OFD>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:Pages><ofd:Page ID="1"/></ofd:Pages></ofd:Document>`,
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

	result, err := svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		t.Fatalf("FirstPageInfo: unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("FirstPageInfo: expected non-nil result")
	}
	if result.MediaBox != nil {
		t.Fatalf("MediaBox = %v, want nil", result.MediaBox)
	}
}

// TestFirstPageInfo_WrongDocType verifies that FirstPageInfo returns
// an error when passed a non-OFD document.
func TestFirstPageInfo_WrongDocType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdfContent := []byte("%PDF-1.7\n1 0 obj\n<<>>\nendobj\n")
	if err := os.WriteFile(path, pdfContent, 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	pdfSvc := pdf.NewService()
	pdfDoc, err := pdfSvc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
	if err != nil {
		t.Fatalf("open PDF: %v", err)
	}
	t.Cleanup(func() { _ = pdfDoc.Close() })

	ofdSvc := NewService()
	_, err = ofdSvc.FirstPageInfo(context.Background(), pdfDoc)
	if err == nil {
		t.Fatalf("FirstPageInfo with PDF doc: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported document type") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "unsupported document type")
	}
}

// TestFirstPageInfo_HelloWorld verifies FirstPageInfo using the real test fixture.
func TestFirstPageInfo_HelloWorld(t *testing.T) {
	const helloPath = "../../testdata/ofd/test_core_helloworld.ofd"
	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: helloPath})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	result, err := svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		t.Fatalf("FirstPageInfo: %v", err)
	}
	if result == nil {
		t.Fatal("FirstPageInfo: expected non-nil result")
	}
	if result.MediaBox == nil {
		t.Fatal("FirstPageInfo: expected MediaBox from helloworld.ofd PhysicalBox")
	}
	if len(result.MediaBox) != 4 {
		t.Fatalf("MediaBox len = %d, want 4", len(result.MediaBox))
	}
	t.Logf("helloworld.ofd MediaBox: %v", result.MediaBox)
}

// TestFirstPageInfo_Multipage verifies FirstPageInfo using the real multipage fixture.
func TestFirstPageInfo_Multipage(t *testing.T) {
	const multiPath = "../../testdata/ofd/test_core_multipage.ofd"
	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: multiPath})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	result, err := svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		t.Fatalf("FirstPageInfo: %v", err)
	}
	if result == nil {
		t.Fatal("FirstPageInfo: expected non-nil result")
	}
	if result.MediaBox == nil {
		t.Fatal("FirstPageInfo: expected MediaBox from multipage.ofd PhysicalBox")
	}
	t.Logf("multipage.ofd MediaBox: %v", result.MediaBox)
}

// TestParsePhysicalBox tests parsePhysicalBox with various inputs.
func TestParsePhysicalBox(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		want    []float64
		wantNil bool
	}{
		{
			name: "standard A4",
			xml:  `<?xml version="1.0"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:CommonData><ofd:PageArea><ofd:PhysicalBox>0 0 210 297</ofd:PhysicalBox></ofd:PageArea></ofd:CommonData></ofd:Document>`,
			want: []float64{0, 0, 210, 297},
		},
		{
			name: "A5 landscape",
			xml:  `<?xml version="1.0"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:CommonData><ofd:PageArea><ofd:PhysicalBox>0 0 148 210</ofd:PhysicalBox></ofd:PageArea></ofd:CommonData></ofd:Document>`,
			want: []float64{0, 0, 148, 210},
		},
		{
			name:    "no PhysicalBox",
			xml:     `<?xml version="1.0"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:Pages/></ofd:Document>`,
			wantNil: true,
		},
		{
			name:    "empty PhysicalBox",
			xml:     `<?xml version="1.0"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:CommonData><ofd:PageArea><ofd:PhysicalBox></ofd:PhysicalBox></ofd:PageArea></ofd:CommonData></ofd:Document>`,
			wantNil: true,
		},
		{
			name:    "malformed PhysicalBox",
			xml:     `<?xml version="1.0"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:CommonData><ofd:PageArea><ofd:PhysicalBox>not numbers</ofd:PhysicalBox></ofd:PageArea></ofd:CommonData></ofd:Document>`,
			wantNil: true,
		},
		{
			name:    "incomplete PhysicalBox",
			xml:     `<?xml version="1.0"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:CommonData><ofd:PageArea><ofd:PhysicalBox>0 0 210</ofd:PhysicalBox></ofd:PageArea></ofd:CommonData></ofd:Document>`,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parsePhysicalBox([]byte(tt.xml))
			if tt.wantNil {
				if got != nil {
					t.Fatalf("parsePhysicalBox = %v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("parsePhysicalBox = nil, want %v", tt.want)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(MediaBox) = %d, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("MediaBox[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestServiceOpenAndInfo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofd.cn/2016/F最低配"><ofd:Pages><ofd:Page ID="1"/></ofd:Pages></ofd:Document>`,
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
	if info.DeclaredVersion != "1.0" {
		t.Fatalf("declared version = %q, want 1.0", info.DeclaredVersion)
	}
	if info.PageCount != 1 {
		t.Fatalf("page count = %d, want 1", info.PageCount)
	}
}

func TestServiceValidateValidOFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
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
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version><DocRoot>Doc_99/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": "<document/>",
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
	if report.Errors[0] != `DocRoot "Doc_99/Document.xml" points to a non-existent file` {
		t.Fatalf("error = %q, want %q", report.Errors[0], `DocRoot "Doc_99/Document.xml" points to a non-existent file`)
	}
}

func TestServiceValidateDocRootMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version></ofd>`,
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write OFD: %v", err)
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
	if report.Errors[0] != "DocRoot element is missing or empty in OFD.xml" {
		t.Fatalf("error = %q, want %q", report.Errors[0], "DocRoot element is missing or empty in OFD.xml")
	}
}

func TestServiceValidateDocRootPointsToNonexistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version><DocRoot>NonExistent/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write OFD: %v", err)
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
	if !strings.Contains(report.Errors[0], "points to a non-existent file") {
		t.Fatalf("error = %q, want contains %q", report.Errors[0], "points to a non-existent file")
	}
}

func TestServiceValidateSealMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "withseal.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": "<document><Pages><Page ID=\"1\"/></Pages></document>",
		"Doc_0/Signs/Signatures.xml": `<?xml version="1.0" encoding="UTF-8"?>
<Signatures>
  <Signature ID="1" BaseLoc="Doc_0/Signs/Sign_0/Signature.xml">
    <SignedInfo>
      <Provider ProviderName="TestProvider"/>
      <Seal><BaseLoc>/Doc_0/Signs/Sign_0/Seal.esl</BaseLoc></Seal>
    </SignedInfo>
  </Signature>
</Signatures>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write OFD: %v", err)
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
		t.Fatalf("valid = false, want true (seal issues are warnings, not errors)")
	}
	if len(report.Warnings) == 0 {
		t.Fatal("warnings is empty, want seal missing warning")
	}
	found := false
	for _, w := range report.Warnings {
		if strings.Contains(w, "Seal.esl not found") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("warnings = %v, want one containing 'Seal.esl not found'", report.Warnings)
	}
}

func TestServiceValidateResourceMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "withres.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": "<document><Pages><Page ID=\"1\"/></Pages></document>",
		"Doc_0/PublicRes.xml": `<?xml version="1.0" encoding="UTF-8"?>
<Resources xmlns="http://www.ofdspec.org/2016">
  <Font ID="10" FontName="TestFont">
    <FontFile>Doc_0/Res/MissingFont.otf</FontFile>
  </Font>
</Resources>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write OFD: %v", err)
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
		t.Fatalf("valid = false, want true (resource issues are warnings, not errors)")
	}
	if len(report.Warnings) == 0 {
		t.Fatal("warnings is empty, want font missing warning")
	}
	found := false
	for _, w := range report.Warnings {
		if strings.Contains(w, "font") && strings.Contains(w, "file not found") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("warnings = %v, want one containing 'font ... file not found'", report.Warnings)
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

func TestServiceExtractTextEmptyOFD(t *testing.T) {
	// An OFD with no pages should return empty text without error.
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016" Version="1.0" DocType="OFD"><ofd:DocBody><ofd:DocRoot>Doc_0/Document.xml</ofd:DocRoot></ofd:DocBody></ofd:OFD>`,
		"Doc_0/Document.xml": `<ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:CommonData/><ofd:Pages></ofd:Pages></ofd:Document>`,
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

	result, err := svc.ExtractText(context.Background(), d)
	if err != nil {
		t.Fatalf("ExtractText unexpected error: %v", err)
	}
	if result.Text != "" {
		t.Fatalf("ExtractText: expected empty text, got %q", result.Text)
	}
}

func TestServiceExtractTextHelloWorld(t *testing.T) {
	const helloPath = "../../testdata/ofd/test_core_helloworld.ofd"
	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: helloPath})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	result, err := svc.ExtractText(context.Background(), d)
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	if result.Text == "" {
		t.Fatal("ExtractText: expected non-empty text for hello world OFD")
	}
	t.Logf("extracted: %q", result.Text)
}

func TestServiceExtractTextKeywordSearch(t *testing.T) {
	const kwPath = "../../testdata/ofd/test_feat_keyword_search.ofd"
	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: kwPath})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	result, err := svc.ExtractText(context.Background(), d)
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}
	if result.Text == "" {
		t.Fatal("ExtractText: expected non-empty text for keyword search OFD")
	}
	t.Logf("extracted %d chars", len(result.Text))
}

// TestServiceExtractTextMultiPage verifies that ExtractText collects text from
// all pages in a multi-page OFD document and concatenates them in page order.
func TestServiceExtractTextMultiPage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016" Version="1.0"><ofd:DocBody><ofd:DocRoot>Doc_0/Document.xml</ofd:DocRoot></ofd:DocBody></ofd:OFD>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofdspec.org/2016"><ofd:Pages>` +
			`<ofd:Page ID="1" BaseLoc="Pages/Page_0/Content.xml"/>` +
			`<ofd:Page ID="2" BaseLoc="Pages/Page_1/Content.xml"/>` +
			`<ofd:Page ID="3" BaseLoc="Pages/Page_2/Content.xml"/>` +
			`</ofd:Pages></ofd:Document>`,
		"Doc_0/Pages/Page_0/Content.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Content xmlns:ofd="http://www.ofdspec.org/2016"><ofd:Page><ofd:TextObject><ofd:TextCode X="0" Y="0">Alpha</ofd:TextCode></ofd:TextObject></ofd:Page></ofd:Content>`,
		"Doc_0/Pages/Page_1/Content.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Content xmlns:ofd="http://www.ofdspec.org/2016"><ofd:Page><ofd:TextObject><ofd:TextCode X="0" Y="0">Beta</ofd:TextCode></ofd:TextObject></ofd:Page></ofd:Content>`,
		"Doc_0/Pages/Page_2/Content.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Content xmlns:ofd="http://www.ofdspec.org/2016"><ofd:Page><ofd:TextObject><ofd:TextCode X="0" Y="0">Gamma</ofd:TextCode></ofd:TextObject></ofd:Page></ofd:Content>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write multi-page OFD: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: path})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	result, err := svc.ExtractText(context.Background(), d)
	if err != nil {
		t.Fatalf("ExtractText: %v", err)
	}

	expected := "Alpha\nBeta\nGamma"
	if result.Text != expected {
		t.Fatalf("ExtractText = %q, want %q", result.Text, expected)
	}
}

func TestServiceInfoMultiPage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofd.cn/2016/F最低配"><ofd:Pages><ofd:Page ID="1"/><ofd:Page ID="2"/><ofd:Page ID="3"/></ofd:Pages></ofd:Document>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write multi-page OFD: %v", err)
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

	if info.PageCount != 3 {
		t.Fatalf("page count = %d, want 3", info.PageCount)
	}
}

func TestServiceInfoNoDocRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nodocroot.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version></ofd>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofd.cn/2016/F最低配"><ofd:Pages><ofd:Page ID="1"/></ofd:Pages></ofd:Document>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write OFD without DocRoot: %v", err)
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

	if info.DeclaredVersion != "1.0" {
		t.Fatalf("declared version = %q, want 1.0", info.DeclaredVersion)
	}
	if info.PageCount != 0 {
		t.Fatalf("page count = %d, want 0 (graceful degradation)", info.PageCount)
	}
}

func TestServiceInfoNoDocumentXml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nodoc.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write OFD without Document.xml: %v", err)
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

	if info.PageCount != 0 {
		t.Fatalf("page count = %d, want 0 (graceful degradation)", info.PageCount)
	}
}

func TestServiceInfoMalformedDocumentXml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": `not valid xml <>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write OFD with malformed Document.xml: %v", err)
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

	if info.PageCount != 0 {
		t.Fatalf("page count = %d, want 0 (graceful degradation)", info.PageCount)
	}
}

func TestServiceInfoMissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noversion.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofd.cn/2016/F最低配"><ofd:Pages><ofd:Page ID="1"/></ofd:Pages></ofd:Document>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write OFD without version: %v", err)
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

	if info.DeclaredVersion != "" {
		t.Fatalf("declared version = %q, want empty", info.DeclaredVersion)
	}
}

func TestServiceInfoMalformedOfdXml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `not valid xml <>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofd.cn/2016/F最低配"><ofd:Pages><ofd:Page ID="1"/></ofd:Pages></ofd:Document>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write OFD with malformed OFD.xml: %v", err)
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

	if info.DeclaredVersion != "" {
		t.Fatalf("declared version = %q, want empty (graceful degradation)", info.DeclaredVersion)
	}
}

// TestValidateZipSafetyTooManyEntries verifies that validateZipSafety rejects
// a ZIP archive whose entry count exceeds maxZipEntries.
func TestValidateZipSafetyTooManyEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "too_many.zip")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	for i := 0; i <= maxZipEntries; i++ {
		w, err := zw.Create(fmt.Sprintf("file%d.txt", i))
		if err != nil {
			t.Fatalf("create entry: %v", err)
		}
		if _, err := w.Write([]byte("x")); err != nil {
			t.Fatalf("write entry: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	f.Close()

	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zr.Close()

	if err := validateZipSafety(zr); err == nil {
		t.Fatal("validateZipSafety: expected error for too many entries, got nil")
	}
}

// TestValidateZipSafetyTooLarge verifies that validateZipSafety rejects a ZIP
// archive whose total uncompressed size exceeds maxDecompressedSize.
func TestValidateZipSafetyTooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "too_large.zip")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)

	// Create a fake entry with huge UncompressedSize64 to simulate a ZIP bomb
	// without writing actual gigabytes of data.
	header := &zip.FileHeader{
		Name:   "large.txt",
		Method: zip.Store,
	}
	header.SetModTime(time.Now())
	header.UncompressedSize64 = maxDecompressedSize + 1
	w, err := zw.CreateRaw(header)
	if err != nil {
		t.Fatalf("CreateRaw: %v", err)
	}
	if _, err := w.Write([]byte("")); err != nil {
		t.Fatalf("write raw: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	f.Close()

	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zr.Close()

	if err := validateZipSafety(zr); err == nil {
		t.Fatal("validateZipSafety: expected error for oversized uncompressed data, got nil")
	}
}

// TestValidateZipSafetyOK verifies that validateZipSafety accepts a normal
// ZIP archive with a small number of entries and modest uncompressed size.
func TestValidateZipSafetyOK(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.zip")

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("OFD.xml")
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if _, err := w.Write([]byte("<ofd/>")); err != nil {
		t.Fatalf("write entry: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	f.Close()

	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer zr.Close()

	if err := validateZipSafety(zr); err != nil {
		t.Fatalf("validateZipSafety: unexpected error: %v", err)
	}
}

// TestReadLimitedTruncation verifies that readLimited returns an error when
// the input stream exceeds the configured size limit.
func TestReadLimitedTruncation(t *testing.T) {
	// Create a reader that yields more than maxXMLReadSize bytes.
	largeData := make([]byte, maxXMLReadSize+1)
	rc := &nopCloser{Reader: bytes.NewReader(largeData)}

	_, err := readLimited(rc, maxXMLReadSize, "test.xml")
	if err == nil {
		t.Fatal("readLimited: expected error for truncated data, got nil")
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "truncated")
	}
}

// TestReadLimitedOK verifies that readLimited returns the full data without
// error when the input stream is smaller than the configured size limit.
func TestReadLimitedOK(t *testing.T) {
	data := []byte("<ofd/>")
	rc := &nopCloser{Reader: bytes.NewReader(data)}

	result, err := readLimited(rc, maxXMLReadSize, "test.xml")
	if err != nil {
		t.Fatalf("readLimited: unexpected error: %v", err)
	}
	if !bytes.Equal(result, data) {
		t.Fatalf("result = %q, want %q", result, data)
	}
}

type nopCloser struct {
	io.Reader
}

func (n *nopCloser) Close() error { return nil }

func TestNewPageIterator_BasicIteration(t *testing.T) {
	const multiPath = "../../testdata/ofd/test_core_multipage.ofd"
	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: multiPath})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	info, err := svc.Info(context.Background(), d)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	iter, err := svc.NewPageIterator(context.Background(), d)
	if err != nil {
		t.Fatalf("NewPageIterator: %v", err)
	}

	var pages []doc.PageData
	for {
		pd, err := iter.Next(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next: %v", err)
		}
		pages = append(pages, pd)
	}

	if len(pages) != info.PageCount {
		t.Fatalf("page count = %d, want %d", len(pages), info.PageCount)
	}
	for i, pd := range pages {
		if pd.Number != i+1 {
			t.Fatalf("pages[%d].Number = %d, want %d", i, pd.Number, i+1)
		}
		if pd.ObjRef == "" {
			t.Fatalf("pages[%d].ObjRef is empty", i)
		}
		if len(pd.Content) == 0 {
			t.Fatalf("pages[%d].Content is empty", i)
		}
	}
}

func TestNewPageIterator_Reset(t *testing.T) {
	const multiPath = "../../testdata/ofd/test_core_multipage.ofd"
	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: multiPath})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	iter, err := svc.NewPageIterator(context.Background(), d)
	if err != nil {
		t.Fatalf("NewPageIterator: %v", err)
	}

	if _, err := iter.Next(context.Background()); err != nil {
		t.Fatalf("first Next: %v", err)
	}
	iter.Reset()

	pd, err := iter.Next(context.Background())
	if err != nil {
		t.Fatalf("Next after Reset: %v", err)
	}
	if pd.Number != 1 {
		t.Fatalf("after Reset: Number = %d, want 1", pd.Number)
	}
}

func TestNewNavigator_GoTo(t *testing.T) {
	const multiPath = "../../testdata/ofd/test_core_multipage.ofd"
	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatOFD, Path: multiPath})
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })

	iter, err := svc.NewPageIterator(context.Background(), d)
	if err != nil {
		t.Fatalf("NewPageIterator: %v", err)
	}
	firstPage, err := iter.Next(context.Background())
	if err != nil {
		t.Fatalf("iter.Next: %v", err)
	}
	ref := firstPage.ObjRef

	nav, err := svc.NewNavigator(context.Background(), d)
	if err != nil {
		t.Fatalf("NewNavigator: %v", err)
	}

	pd, err := nav.GoTo(context.Background(), ref)
	if err != nil {
		t.Fatalf("GoTo: %v", err)
	}
	if pd.Number < 1 {
		t.Fatalf("GoTo(%q).Number = %d, want >= 1", ref, pd.Number)
	}
}
