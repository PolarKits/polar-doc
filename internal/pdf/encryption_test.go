package pdf

import (
	"context"
	"os"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
	testfixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

// TestEncryptionAlgorithm_Detection verifies that PDFFeatureSet correctly
// reports encryption state and algorithm for encrypted documents, and
// reports EncryptNone for unencrypted documents.
func TestEncryptionAlgorithm_Detection(t *testing.T) {
	svc := NewService()

	// Non-encrypted fixture: encryption should be absent.
	t.Run("non-encrypted", func(t *testing.T) {
		sample, ok := testfixtures.PDFSampleByKey("core-minimal")
		if !ok {
			t.Fatalf("missing PDF sample core-minimal")
		}
		d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
		if err != nil {
			t.Fatalf("open PDF: %v", err)
		}
		defer d.Close()

		features, err := svc.DocumentFeatures(context.Background(), d)
		if err != nil {
			t.Fatalf("DocumentFeatures: %v", err)
		}
		if features.IsEncrypted {
			t.Fatalf("IsEncrypted = true, expected false for non-encrypted fixture")
		}
		if features.EncryptionAlgorithm != EncryptNone {
			t.Fatalf("EncryptionAlgorithm = %v, expected EncryptNone for non-encrypted fixture", features.EncryptionAlgorithm)
		}
	})

		// Encrypted fixture: should detect encryption and map to a known algorithm.
		// The registered feature-encrypted fixture (test_feat_encrypted_v1.5.pdf)
		// is not present on disk, so we fall back to an existing encrypted PDF
		// from the pmaupin corpus (V=2, R=3 → EncryptRC4_128).
		t.Run("encrypted", func(t *testing.T) {
			// Use pmaupin_6e122f.pdf which has /Encrypt with /V 2 /R 3.
			sample, ok := testfixtures.PDFSampleByKey("core-minimal")
			if !ok {
				t.Fatalf("missing PDF sample core-minimal")
			}
			path := sample.Path()
			path = path[:len(path)-len("test_core_minimal_v1.5.pdf")] + "pmaupin_6e122f.pdf"

			d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
			if err != nil {
				t.Fatalf("open PDF: %v", err)
			}
			defer d.Close()

			features, err := svc.DocumentFeatures(context.Background(), d)
			if err != nil {
				t.Fatalf("DocumentFeatures: %v", err)
			}
			if !features.IsEncrypted {
				t.Fatalf("IsEncrypted = false, expected true for encrypted fixture")
			}
			if features.EncryptionAlgorithm == EncryptNone {
				t.Fatalf("EncryptionAlgorithm = EncryptNone, expected a known algorithm for encrypted fixture")
			}
			if features.EncryptionAlgorithm == EncryptUnknown {
				t.Fatalf("EncryptionAlgorithm = EncryptUnknown, expected a known algorithm for encrypted fixture")
			}
		})
}

// TestDetectEncryptionAlgorithm_Logic exercises detectEncryptionAlgorithm
// directly with synthetic dictionaries covering the /V and /R combinations
// specified in ISO 32000-1 Table 3.18 and ISO 32000-2 §7.6.
func TestDetectEncryptionAlgorithm_Logic(t *testing.T) {
	tests := []struct {
		name   string
		dict   PDFDict
		want   EncryptionAlgorithm
	}{
		{
			name: "empty-dict",
			dict: PDFDict{},
			want: EncryptNone,
		},
		{
			name: "V1_R2",
			dict: PDFDict{"V": PDFInteger(1), "R": PDFInteger(2)},
			want: EncryptRC4_40,
		},
		{
			name: "V1_R3",
			dict: PDFDict{"V": PDFInteger(1), "R": PDFInteger(3)},
			want: EncryptRC4_128,
		},
		{
			name: "V2_R3",
			dict: PDFDict{"V": PDFInteger(2), "R": PDFInteger(3)},
			want: EncryptRC4_128,
		},
		{
			name: "V4_R4_no_CF",
			dict: PDFDict{"V": PDFInteger(4), "R": PDFInteger(4)},
			want: EncryptRC4_128,
		},
		{
			name: "V4_R4_RC4_StdCF",
			dict: PDFDict{
				"V": PDFInteger(4),
				"R": PDFInteger(4),
				"CF": PDFDict{
					"StdCF": PDFDict{"CFM": PDFName("V2")},
				},
			},
			want: EncryptRC4_128,
		},
		{
			name: "V4_R4_AES_StdCF",
			dict: PDFDict{
				"V": PDFInteger(4),
				"R": PDFInteger(4),
				"CF": PDFDict{
					"StdCF": PDFDict{"CFM": PDFName("AESV2")},
				},
			},
			want: EncryptAES_128,
		},
		{
			name: "V5_R5",
			dict: PDFDict{"V": PDFInteger(5), "R": PDFInteger(5)},
			want: EncryptAES_256,
		},
		{
			name: "V5_R6",
			dict: PDFDict{"V": PDFInteger(5), "R": PDFInteger(6)},
			want: EncryptAES_256,
		},
		{
			name: "V3_R2_unexpected",
			dict: PDFDict{"V": PDFInteger(3), "R": PDFInteger(2)},
			want: EncryptUnknown,
		},
		{
			name: "V4_R5_unexpected",
			dict: PDFDict{"V": PDFInteger(4), "R": PDFInteger(5)},
			want: EncryptUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Passing nil for *os.File is safe here because the input is already
			// a PDFDict; no indirect resolution is required.
			got := detectEncryptionAlgorithm(nil, tc.dict)
			if got != tc.want {
				t.Fatalf("detectEncryptionAlgorithm = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestDetectEncryptionAlgorithm_RefResolution verifies that a PDFRef is
// resolved to the underlying object and its fields are read correctly.
func TestDetectEncryptionAlgorithm_RefResolution(t *testing.T) {
	// Open an existing encrypted PDF and pass the Encrypt ref object.
	sample, ok := testfixtures.PDFSampleByKey("core-minimal")
	if !ok {
		t.Fatalf("missing PDF sample core-minimal")
	}
	path := sample.Path()
	path = path[:len(path)-len("test_core_minimal_v1.5.pdf")] + "pmaupin_6e122f.pdf"

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	// pmaupin_6e122f.pdf has /Encrypt 940 0 R and /V 2 /R 3 inside.
	ref := PDFRef{ObjNum: 940, GenNum: 0}
	got := detectEncryptionAlgorithm(f, ref)
	if got != EncryptRC4_128 {
		t.Fatalf("detectEncryptionAlgorithm via ref = %v, want EncryptRC4_128", got)
	}
}

// TestDetectEncryptionAlgorithm_InvalidRef verifies that an unresolvable
// reference returns EncryptNone instead of crashing.
func TestDetectEncryptionAlgorithm_InvalidRef(t *testing.T) {
	sample, ok := testfixtures.PDFSampleByKey("core-minimal")
	if !ok {
		t.Fatalf("missing PDF sample core-minimal")
	}
	path := sample.Path()
	path = path[:len(path)-len("test_core_minimal_v1.5.pdf")] + "pmaupin_6e122f.pdf"

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer f.Close()

	// Object 9999 does not exist in the file.
	ref := PDFRef{ObjNum: 9999, GenNum: 0}
	got := detectEncryptionAlgorithm(f, ref)
	if got != EncryptNone {
		t.Fatalf("detectEncryptionAlgorithm invalid ref = %v, want EncryptNone", got)
	}
}
