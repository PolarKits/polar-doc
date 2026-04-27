package ofd

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestParseAnnotationsXML_Invoice(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_feat_invoice.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	doc, err := ParseAnnotationsXML(zr.File)
	if err != nil {
		t.Fatalf("ParseAnnotationsXML: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil result")
	}

	if len(doc.Pages) == 0 {
		t.Fatal("expected at least one page annotation entry")
	}

	page := doc.Pages[0]
	if page.PageID != 1 {
		t.Errorf("PageID = %d, want 1", page.PageID)
	}
	if page.FilePath == "" {
		t.Error("FilePath should not be empty")
	}
	t.Logf("Page annotation: PageID=%d FilePath=%s", page.PageID, page.FilePath)
}

func TestParseAnnotationsXML_Transparency(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_feat_transparency.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	doc, err := ParseAnnotationsXML(zr.File)
	if err != nil {
		t.Fatalf("ParseAnnotationsXML: %v", err)
	}
	if doc == nil {
		t.Fatal("expected non-nil result")
	}

	if len(doc.Pages) == 0 {
		t.Fatal("expected at least one page annotation entry")
	}
	t.Logf("Transparency annotations: %d page(s)", len(doc.Pages))
}

func TestParseAnnotationsXML_NotFound(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.Close()

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("create empty zip: %v", err)
	}

	doc, err := ParseAnnotationsXML(zr.File)
	if err != nil {
		t.Fatalf("ParseAnnotationsXML should not error on missing files: %v", err)
	}
	if doc != nil {
		t.Error("expected nil result for OFD without Annotations.xml")
	}
}

func TestParsePageAnnotations_Watermark(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:PageAnnot xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:Annot Type="Watermark" Creator="OFD R&amp;W" LastModDate="2024-09-29" ID="5">
    <ofd:Appearance Boundary="0 0 420 297">
      <ofd:TextObject Boundary="0 0 420 297" Font="6" Size="5.644" ID="7" Fill="true" Alpha="76">
        <ofd:TextCode X="-11.281" Y="5.644" DeltaX="5.641 5.641 5.641">测试水印</ofd:TextCode>
      </ofd:TextObject>
    </ofd:Appearance>
  </ofd:Annot>
</ofd:PageAnnot>`)

	annotations, err := ParsePageAnnotations(data)
	if err != nil {
		t.Fatalf("ParsePageAnnotations: %v", err)
	}

	if len(annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(annotations))
	}

	annot := annotations[0]
	if annot.ID != 5 {
		t.Errorf("ID = %d, want 5", annot.ID)
	}
	if annot.Type != AnnotationTypeWatermark {
		t.Errorf("Type = %q, want %q", annot.Type, AnnotationTypeWatermark)
	}
	if annot.Creator != "OFD R&W" {
		t.Errorf("Creator = %q, want %q", annot.Creator, "OFD R&W")
	}
	if annot.Modified != "2024-09-29" {
		t.Errorf("Modified = %q, want %q", annot.Modified, "2024-09-29")
	}
	if len(annot.Boundary) != 4 {
		t.Errorf("Boundary len = %d, want 4", len(annot.Boundary))
	}
}

func TestParsePageAnnotations_Stamp(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:PageAnnot xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:Annot Type="Stamp" ID="117" Subtype="SignatureInFile">
    <ofd:Parameters>
      <ofd:Parameter Name="fp.NativeSign">original_invoice</ofd:Parameter>
    </ofd:Parameters>
    <ofd:Appearance Boundary="4.5 104 115 20"/>
  </ofd:Annot>
</ofd:PageAnnot>`)

	annotations, err := ParsePageAnnotations(data)
	if err != nil {
		t.Fatalf("ParsePageAnnotations: %v", err)
	}

	if len(annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(annotations))
	}

	annot := annotations[0]
	if annot.ID != 117 {
		t.Errorf("ID = %d, want 117", annot.ID)
	}
	if annot.Type != AnnotationTypeStamp {
		t.Errorf("Type = %q, want %q", annot.Type, AnnotationTypeStamp)
	}
	if annot.Subtype != "SignatureInFile" {
		t.Errorf("Subtype = %q, want %q", annot.Subtype, "SignatureInFile")
	}
	if annot.Parameters["fp.NativeSign"] != "original_invoice" {
		t.Errorf("Parameters[fp.NativeSign] = %q, want %q", annot.Parameters["fp.NativeSign"], "original_invoice")
	}
}

func TestParsePageAnnotations_Multiple(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:PageAnnot xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:Annot Type="Watermark" ID="5"><ofd:Appearance Boundary="0 0 100 100"/></ofd:Annot>
  <ofd:Annot Type="Watermark" ID="9"><ofd:Appearance Boundary="0 0 200 200"/></ofd:Annot>
</ofd:PageAnnot>`)

	annotations, err := ParsePageAnnotations(data)
	if err != nil {
		t.Fatalf("ParsePageAnnotations: %v", err)
	}

	if len(annotations) != 2 {
		t.Fatalf("expected 2 annotations, got %d", len(annotations))
	}
	if annotations[0].ID != 5 || annotations[1].ID != 9 {
		t.Errorf("annotation IDs: got %d, %d, want 5, 9", annotations[0].ID, annotations[1].ID)
	}
}

func TestAnnotationType_Constants(t *testing.T) {
	if AnnotationTypeHighlight != "Highlight" {
		t.Errorf("AnnotationTypeHighlight = %q, want %q", AnnotationTypeHighlight, "Highlight")
	}
	if AnnotationTypeStamp != "Stamp" {
		t.Errorf("AnnotationTypeStamp = %q, want %q", AnnotationTypeStamp, "Stamp")
	}
	if AnnotationTypeWatermark != "Watermark" {
		t.Errorf("AnnotationTypeWatermark = %q, want %q", AnnotationTypeWatermark, "Watermark")
	}
	if AnnotationTypeLink != "Link" {
		t.Errorf("AnnotationTypeLink = %q, want %q", AnnotationTypeLink, "Link")
	}
	if AnnotationTypeText != "Text" {
		t.Errorf("AnnotationTypeText = %q, want %q", AnnotationTypeText, "Text")
	}
}