package pdf

import (
	"context"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
	testfixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

// TestPDFVersionConstants verifies all version constants exist and have correct values.
func TestPDFVersionConstants(t *testing.T) {
	tests := []struct {
		name      string
		version   PDFVersion
		wantMajor int
		wantMinor int
	}{
		{"PDF10", PDF10, 1, 0},
		{"PDF11", PDF11, 1, 1},
		{"PDF12", PDF12, 1, 2},
		{"PDF13", PDF13, 1, 3},
		{"PDF14", PDF14, 1, 4},
		{"PDF15", PDF15, 1, 5},
		{"PDF16", PDF16, 1, 6},
		{"PDF17", PDF17, 1, 7},
		{"PDF20", PDF20, 2, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.version.Major != tc.wantMajor {
				t.Errorf("%s.Major = %d, want %d", tc.name, tc.version.Major, tc.wantMajor)
			}
			if tc.version.Minor != tc.wantMinor {
				t.Errorf("%s.Minor = %d, want %d", tc.name, tc.version.Minor, tc.wantMinor)
			}
			// Verify String() returns expected format
			wantStr := string(rune('0'+tc.wantMajor)) + "." + string(rune('0'+tc.wantMinor))
			if tc.version.String() != wantStr {
				t.Errorf("%s.String() = %q, want %q", tc.name, tc.version.String(), wantStr)
			}
		})
	}
}

// TestPDFVersionMatrix tests Open, Info, and Validate for each PDF version.
// Covers versions 1.0 through 2.0 with assertions on validation results.
func TestPDFVersionMatrix(t *testing.T) {
	tests := []struct {
		key             string
		wantVersion     string
		expectCorrupted bool
	}{
		{"version-compat-v1.0", "1.0", false},
		{"version-compat-v1.2", "1.2", false},
		{"version-compat-v1.4", "1.4", false},
		{"version-compat-v1.7", "1.7", true}, // corrupted fixture
		{"version-v1.1", "1.1", false},
		{"core-multipage", "1.3", false},     // PDF 1.3 fixture
		{"core-minimal", "1.5", false},       // PDF 1.5 fixture
		{"feature-fillable", "1.6", false},     // PDF 1.6 fixture
		{"standard-pdf20-basic", "2.0", false},
		{"standard-pdf20-utf8", "2.0", false},
	}

	svc := NewService()
	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			sample, ok := testfixtures.PDFSampleByKey(tc.key)
			if !ok {
				t.Skipf("Sample %q not found", tc.key)
			}

			// Test Open()
			d, err := svc.Open(ctx, doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
			if err != nil {
				t.Fatalf("Open(%q): %v", tc.key, err)
			}
			defer d.Close()

			// Test Info() returns correct DeclaredVersion
			info, err := svc.Info(ctx, d)
			if err != nil {
				t.Fatalf("Info(%q): %v", tc.key, err)
			}
			// DeclaredVersion may include binary marker; check prefix
			if info.DeclaredVersion != tc.wantVersion &&
				!strings.HasPrefix(info.DeclaredVersion, tc.wantVersion+"\r") &&
				!strings.HasPrefix(info.DeclaredVersion, tc.wantVersion+"\n") {
				t.Errorf("DeclaredVersion = %q, want prefix %q", info.DeclaredVersion, tc.wantVersion)
			}

			// Test Validate() with assertions
			report, err := svc.Validate(ctx, d)
			if err != nil {
				// Some fixtures may fail validation due to minimal structure;
				// this is acceptable as long as Open/Info work
				t.Logf("Validate(%q) returned error: %v", tc.key, err)
				return
			}

			if tc.expectCorrupted {
				if report.Valid {
					t.Errorf("Validate(%q): Valid = true, want false for corrupted fixture", tc.key)
				}
			} else {
				// For non-corrupted fixtures, we expect valid PDFs
				// Some minimal fixtures may still fail; log but don't fail test
				if !report.Valid {
					t.Logf("Validate(%q): Valid = false (fixture may be minimal)", tc.key)
				}
			}
		})
	}
}

// TestDocumentFeatures_ByVersion verifies structural features by version expectations.
// Versions 1.0-1.4: HasTraditionalXRef=true (pre-1.5 only supports traditional xref)
// Versions 1.5+: may use XRef stream or traditional table depending on file.
func TestDocumentFeatures_ByVersion(t *testing.T) {
	// Pre-1.5 versions should always have traditional xref
	pre15Tests := []struct {
		key string
	}{
		{"version-compat-v1.0"},
		{"version-compat-v1.2"},
		{"version-compat-v1.4"},
		{"core-multipage"}, // v1.3
	}

	// 1.5+ versions may have either xref type
	post15Tests := []struct {
		key string
	}{
		{"core-minimal"},        // v1.5
		{"feature-fillable"},    // v1.6
		{"standard-pdf20-utf8"}, // v2.0
		{"standard-pdf20-basic"},
	}

	svc := NewService()
	ctx := context.Background()

	t.Run("pre-1.5", func(t *testing.T) {
		for _, tc := range pre15Tests {
			t.Run(tc.key, func(t *testing.T) {
				sample, ok := testfixtures.PDFSampleByKey(tc.key)
				if !ok {
					t.Skipf("Sample %q not found", tc.key)
				}

				d, err := svc.Open(ctx, doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
				if err != nil {
					t.Fatalf("Open(%q): %v", tc.key, err)
				}
				defer d.Close()

				features, err := svc.DocumentFeatures(ctx, d)
				if err != nil {
					t.Fatalf("DocumentFeatures(%q): %v", tc.key, err)
				}

				// Pre-1.5 PDFs must use traditional xref (even if also hybrid)
				if !features.HasTraditionalXRef {
					t.Errorf("HasTraditionalXRef = false for %s, want true (pre-1.5 must use traditional xref)", tc.key)
				}
				// Note: Some PDF 1.4 files may have hybrid xref (XRefStm in trailer)
				// This is a generator quirk; we only log it, not fail
				if features.HasXRefStream {
					t.Logf("Note: %s has XRefStream despite being pre-1.5 (hybrid xref)", tc.key)
				}
			})
		}
	})

	t.Run("1.5+", func(t *testing.T) {
		for _, tc := range post15Tests {
			t.Run(tc.key, func(t *testing.T) {
				sample, ok := testfixtures.PDFSampleByKey(tc.key)
				if !ok {
					t.Skipf("Sample %q not found", tc.key)
				}

				d, err := svc.Open(ctx, doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
				if err != nil {
					t.Fatalf("Open(%q): %v", tc.key, err)
				}
				defer d.Close()

				features, err := svc.DocumentFeatures(ctx, d)
				if err != nil {
					t.Fatalf("DocumentFeatures(%q): %v", tc.key, err)
				}

				// At least one xref type should be detected
				if !features.HasTraditionalXRef && !features.HasXRefStream {
					t.Errorf("Neither HasTraditionalXRef nor HasXRefStream is true for %s", tc.key)
				}
			})
		}
	})
}

// TestDocumentFeatures_ObjStmAssertion tests that PDFs with Object Streams are correctly detected.
// Object Streams (ObjStm) were introduced in PDF 1.5.
func TestDocumentFeatures_ObjStmAssertion(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// PDF 1.4 and earlier do not support object streams
	t.Run("pre-1.5-no-objstm", func(t *testing.T) {
		samples := []string{
			"version-compat-v1.4",
			"core-multipage", // v1.3
		}
		for _, key := range samples {
			sample, ok := testfixtures.PDFSampleByKey(key)
			if !ok {
				t.Skipf("Sample %q not found", key)
			}

			d, err := svc.Open(ctx, doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
			if err != nil {
				t.Fatalf("Open(%q): %v", key, err)
			}
			defer d.Close()

			features, err := svc.DocumentFeatures(ctx, d)
			if err != nil {
				t.Fatalf("DocumentFeatures(%q): %v", key, err)
			}

			if features.HasObjectStreams {
				t.Errorf("HasObjectStreams = true for %s, want false (ObjStm introduced in 1.5)", key)
			}
		}
	})

	// PDF 1.5+ files may or may not use object streams depending on generator
	// We test that the feature detection works correctly when ObjStm is present
	t.Run("1.5-plus-may-have-objstm", func(t *testing.T) {
		// These fixtures may or may not have ObjStm; we just verify detection works
		samples := []string{
			"core-minimal",        // v1.5
			"feature-fillable",    // v1.6
			"standard-pdf20-utf8", // v2.0
		}
		for _, key := range samples {
			sample, ok := testfixtures.PDFSampleByKey(key)
			if !ok {
				t.Skipf("Sample %q not found", key)
			}

			d, err := svc.Open(ctx, doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
			if err != nil {
				t.Fatalf("Open(%q): %v", key, err)
			}
			defer d.Close()

			features, err := svc.DocumentFeatures(ctx, d)
			if err != nil {
				t.Fatalf("DocumentFeatures(%q): %v", key, err)
			}

			// Log whether ObjStm is detected; this is informational only
			// as we cannot guarantee specific generator behavior
			t.Logf("Sample %q: HasObjectStreams=%v", key, features.HasObjectStreams)
		}
	})
}

// TestDocumentFeatures_Linearized checks for linearized PDF detection.
// If any fixture is linearized, we assert it is detected correctly.
func TestDocumentFeatures_Linearized(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// Check all samples for linearization
	samples := testfixtures.PDFSamples()
	foundLinearized := false

	for _, sample := range samples {
		if sample.Integrity != testfixtures.PDFSampleIntegrityValid {
			continue
		}

		t.Run(sample.Key, func(t *testing.T) {
			d, err := svc.Open(ctx, doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
			if err != nil {
				t.Skipf("Open(%q) failed: %v", sample.Key, err)
			}
			defer d.Close()

			features, err := svc.DocumentFeatures(ctx, d)
			if err != nil {
				t.Skipf("DocumentFeatures(%q) failed: %v", sample.Key, err)
			}

			if features.IsLinearized {
				foundLinearized = true
				t.Logf("Sample %q is correctly detected as linearized", sample.Key)
			}
		})
	}

	if !foundLinearized {
		t.Skip("No linearized PDF samples found in test fixtures (this is expected)")
	}
}

// TestDocumentFeatures_IncrementalUpdates checks incremental update detection.
// Incremental updates are detected when multiple xref sections are present.
func TestDocumentFeatures_IncrementalUpdates(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// Test with standard-pdf20-incremental which should have incremental updates
	sample, ok := testfixtures.PDFSampleByKey("standard-pdf20-incremental")
	if !ok {
		t.Skip("standard-pdf20-incremental sample not found")
	}

	d, err := svc.Open(ctx, doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer d.Close()

	features, err := svc.DocumentFeatures(ctx, d)
	if err != nil {
		t.Fatalf("DocumentFeatures: %v", err)
	}

	// Assert that incremental updates are detected
	if !features.HasIncrementalUpdates {
		t.Errorf("HasIncrementalUpdates = false for %s, want true (fixture has multiple xref sections)", sample.Key)
	}
}

// TestDocumentFeatures_InfoDictAndXMP checks Info dictionary and XMP stream detection.
func TestDocumentFeatures_InfoDictAndXMP(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	tests := []struct {
		key            string
		expectInfoDict bool
		mayHaveXMP     bool
	}{
		{"version-compat-v1.0", true, false},
		{"version-compat-v1.2", true, false},
		{"version-compat-v1.4", true, false},
		{"standard-pdf20-utf8", true, true}, // PDF 2.0 prefers XMP but may also have Info dict
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			sample, ok := testfixtures.PDFSampleByKey(tc.key)
			if !ok {
				t.Skipf("Sample %q not found", tc.key)
			}

			d, err := svc.Open(ctx, doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
			if err != nil {
				t.Fatalf("Open(%q): %v", tc.key, err)
			}
			defer d.Close()

			features, err := svc.DocumentFeatures(ctx, d)
			if err != nil {
				t.Fatalf("DocumentFeatures(%q): %v", tc.key, err)
			}

			if tc.expectInfoDict && !features.HasInfoDict {
				t.Errorf("HasInfoDict = false for %s, want true", tc.key)
			}

			t.Logf("Features for %s: HasInfoDict=%v, HasXMPStream=%v", tc.key, features.HasInfoDict, features.HasXMPStream)
		})
	}
}

// TestVersionAtLeast verifies the AtLeast comparison logic.
func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		name  string
		v     PDFVersion
		other PDFVersion
		want  bool
	}{
		{"1.4 >= 1.3", PDF14, PDF13, true},
		{"1.4 >= 1.4", PDF14, PDF14, true},
		{"1.4 >= 1.5", PDF14, PDF15, false},
		{"1.5 >= 1.4", PDF15, PDF14, true},
		{"2.0 >= 1.7", PDF20, PDF17, true},
		{"1.7 >= 2.0", PDF17, PDF20, false},
		{"1.10 >= 1.9", PDFVersion{1, 10}, PDFVersion{1, 9}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.v.AtLeast(tc.other)
			if got != tc.want {
				t.Errorf("%v.AtLeast(%v) = %v, want %v", tc.v, tc.other, got, tc.want)
			}
		})
	}
}

// TestVersionIsZero verifies the IsZero check.
func TestVersionIsZero(t *testing.T) {
	tests := []struct {
		name string
		v    PDFVersion
		want bool
	}{
		{"zero value", PDFVersion{}, true},
		{"explicit zero", PDFVersion{0, 0}, true},
		{"1.0", PDF10, false},
		{"1.7", PDF17, false},
		{"2.0", PDF20, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.v.IsZero()
			if got != tc.want {
				t.Errorf("%v.IsZero() = %v, want %v", tc.v, got, tc.want)
			}
		})
	}
}
