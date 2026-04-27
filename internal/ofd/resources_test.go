package ofd

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestParseResourcesXML_HelloWorld(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_core_helloworld.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	res, err := ParseResourcesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseResourcesXML: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}

	if len(res.Fonts) == 0 {
		t.Fatal("expected at least one font")
	}

	font := res.Fonts[0]
	if font.ID != 3 {
		t.Errorf("Font ID = %d, want 3", font.ID)
	}
	if font.FontName != "宋体" {
		t.Errorf("FontName = %q, want %q", font.FontName, "宋体")
	}
}

func TestParseResourcesXML_Multipage(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_core_multipage.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	res, err := ParseResourcesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseResourcesXML: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}

	if len(res.Fonts) == 0 {
		t.Fatal("expected fonts from PublicRes.xml")
	}
	if len(res.MultiMedias) == 0 {
		t.Fatal("expected multimedia from DocumentRes.xml")
	}

	hasImage := false
	for _, mm := range res.MultiMedias {
		if mm.Type == "Image" {
			hasImage = true
			if mm.FilePath == "" {
				t.Error("Image MultiMedia missing FilePath")
			}
		}
	}
	if !hasImage {
		t.Error("expected at least one Image multimedia")
	}
}

func TestParseResourcesXML_NotFound(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.Close()

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("create empty zip: %v", err)
	}

	res, err := ParseResourcesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseResourcesXML should not error on missing files: %v", err)
	}
	if res != nil {
		t.Error("expected nil result for OFD without resources")
	}
}

func TestFontInfo_Attributes(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_feat_pattern.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	res, err := ParseResourcesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseResourcesXML: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}

	boldFont := false
	for _, f := range res.Fonts {
		if f.Bold {
			boldFont = true
			if f.FontName == "" {
				t.Error("bold font missing FontName")
			}
		}
		if f.FixedWidth {
			t.Logf("FixedWidth font: ID=%d FontName=%q", f.ID, f.FontName)
		}
	}
	if !boldFont {
		t.Error("expected at least one bold font")
	}
}

func TestMultiMediaInfo_Attributes(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_feat_images.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	res, err := ParseResourcesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseResourcesXML: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}

	if len(res.MultiMedias) == 0 {
		t.Fatal("expected multimedia entries")
	}

	for _, mm := range res.MultiMedias {
		if mm.ID == 0 {
			t.Error("MultiMedia ID should not be zero")
		}
		if mm.Type == "" {
			t.Error("MultiMedia Type should not be empty")
		}
		if mm.FilePath == "" {
			t.Errorf("MultiMedia ID=%d missing FilePath", mm.ID)
		}
		t.Logf("MultiMedia: ID=%d Type=%q Format=%q File=%q", mm.ID, mm.Type, mm.Format, mm.FilePath)
	}
}

func TestDocumentResources_Empty(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_feat_complex_layout.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	res, err := ParseResourcesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseResourcesXML: %v", err)
	}
	if res != nil {
		t.Error("expected nil result when no PublicRes/DocumentRes exists")
	}
}