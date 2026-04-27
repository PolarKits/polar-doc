package ofd

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestParseSignaturesXML_Multipage(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_core_multipage.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	sigs, err := ParseSignaturesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseSignaturesXML: %v", err)
	}

	if len(sigs) == 0 {
		t.Fatal("expected at least one signature")
	}

	for i, sig := range sigs {
		t.Logf("Signature %d: ID=%d Provider=%q Version=%q Company=%q",
			i, sig.ID, sig.ProviderName, sig.ProviderVersion, sig.ProviderCompany)
	}
}

func TestParseSignaturesXML_Invoice(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_feat_invoice.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	sigs, err := ParseSignaturesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseSignaturesXML: %v", err)
	}

	if len(sigs) == 0 {
		t.Fatal("expected at least one signature")
	}

	sig := sigs[0]
	if sig.ID == 0 {
		t.Error("Signature ID should not be zero")
	}
}

func TestParseSignaturesXML_NotFound(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	zw.Close()

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("create empty zip: %v", err)
	}

	sigs, err := ParseSignaturesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseSignaturesXML should not error on missing files: %v", err)
	}
	if sigs != nil {
		t.Error("expected nil result for OFD without Signatures.xml")
	}
}

func TestParseSignaturesXML_Signature(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_feat_signature.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	sigs, err := ParseSignaturesXML(zr.File)
	if err != nil {
		t.Fatalf("ParseSignaturesXML: %v", err)
	}

	if len(sigs) == 0 {
		t.Fatal("expected at least one signature")
	}

	sig := sigs[0]
	if sig.ID != 1 {
		t.Errorf("Signature ID = %d, want 1", sig.ID)
	}
}

func TestSignatureInfo_Fields(t *testing.T) {
	sig := SignatureInfo{
		ID:                42,
		ProviderName:      "TestProvider",
		ProviderVersion:   "1.0",
		ProviderCompany:   "TestCorp",
		SignatureMethod:   "1.2.156.10197.1.501",
		SignatureDateTime: "20201010065840.923Z",
		StampAnnotID:      1,
		StampAnnotPageRef: 5,
		StampAnnotBoundary: []float64{10, 20, 30, 40},
		SealBaseLoc:       "/Doc_0/Signs/Sign_0/Seal.esl",
		SignedValuePath:   "/Doc_0/Signs/Sign_0/SignedValue.dat",
	}

	if sig.ID != 42 {
		t.Errorf("ID = %d, want 42", sig.ID)
	}
	if sig.ProviderName != "TestProvider" {
		t.Errorf("ProviderName = %q, want %q", sig.ProviderName, "TestProvider")
	}
	if len(sig.StampAnnotBoundary) != 4 {
		t.Errorf("StampAnnotBoundary len = %d, want 4", len(sig.StampAnnotBoundary))
	}
}

func TestParseSignaturesXML_Debug(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Signatures xmlns:ofd="http://www.ofdspec.org/2016">
  <ofd:Signature ID="1" BaseLoc="/Doc_0/Signs/Sign_0/Signature.xml"/>
</ofd:Signatures>`)

	sigs, err := parseSignaturesXMLData(data)
	if err != nil {
		t.Fatalf("parseSignaturesXMLData: %v", err)
	}
	if len(sigs) != 1 {
		t.Fatalf("expected 1 signature, got %d", len(sigs))
	}
	t.Logf("Parsed signature: ID=%d", sigs[0].ID)
}

func TestParseSealESL_Multipage(t *testing.T) {
	ofdPath := "../../testdata/ofd/test_core_multipage.ofd"
	zr, err := zip.OpenReader(ofdPath)
	if err != nil {
		t.Fatalf("open OFD: %v", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		name := strings.TrimPrefix(f.Name, "./")
		if name == "Doc_0/Signs/Sign_0/Seal.esl" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open Seal.esl: %v", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				t.Fatalf("read Seal.esl: %v", err)
			}

			result, err := ParseSealESL(data)
			if err != nil {
				t.Fatalf("ParseSealESL: %v", err)
			}
			if result.Seal.Version == "" {
				t.Error("Seal.Version should not be empty")
			}
			if result.Seal.Width == 0 || result.Seal.Height == 0 {
				t.Errorf("Seal dimensions should be non-zero: %dx%d", result.Seal.Width, result.Seal.Height)
			}
			if result.Seal.Picture.Format != "PNG" {
				t.Errorf("Seal.Picture.Format = %q, want PNG", result.Seal.Picture.Format)
			}
			t.Logf("Seal: Version=%s Size=%dx%d Picture=%dx%d",
				result.Seal.Version, result.Seal.Width, result.Seal.Height,
				result.Seal.Picture.Width, result.Seal.Picture.Height)
			return
		}
	}
	t.Skip("Seal.esl not found in test file")
}

func TestParseSealESL_InvalidMagic(t *testing.T) {
	data := []byte("NOT_A_SEAL_FILE")
	_, err := ParseSealESL(data)
	if err == nil {
		t.Error("expected error for invalid magic bytes")
	}
}

func TestParseSealESL_TooShort(t *testing.T) {
	data := []byte{0x45, 0x53, 0x01}
	_, err := ParseSealESL(data)
	if err == nil {
		t.Error("expected error for too-short data")
	}
}