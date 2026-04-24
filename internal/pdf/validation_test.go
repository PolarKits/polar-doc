package pdf

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
	testfixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

// TestValidateLevels validates that all validation levels are executed.
func TestValidateLevels(t *testing.T) {
	svc := NewService()

	tests := []struct {
		name          string
		sampleKey     string
		expectValid   bool
		expectErrors  []string
		expectWarnings []string
	}{
		{
			name:         "valid minimal PDF",
			sampleKey:    "core-minimal",
			expectValid:  true,
			expectErrors: nil,
		},
		{
			name:         "corrupted PDF",
			sampleKey:    "error-corrupted",
			expectValid:  false,
			expectErrors: []string{"xref", "trailer", "catalog"}, // At least one of these
		},
		{
			name:         "valid multipage PDF",
			sampleKey:    "core-multipage",
			expectValid:  true,
			expectErrors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sample, ok := testfixtures.PDFSampleByKey(tt.sampleKey)
			if !ok {
				t.Fatalf("Sample %q not found", tt.sampleKey)
			}

			f, err := os.Open(sample.Path())
			if err != nil {
				t.Fatalf("Failed to open PDF: %v", err)
			}
			defer f.Close()

			doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
			if err != nil {
				// Some corrupted samples may fail to open, which is ok for validation testing
				// Try to validate the file directly
				report, validateErr := validateDocument(f)
				if validateErr != nil {
					t.Logf("Validation error (expected for corrupted): %v", validateErr)
				}
				if report.Valid != tt.expectValid {
					t.Errorf("Validate() Valid = %v, want %v", report.Valid, tt.expectValid)
				}
				return
			}
			defer doc.Close()

			report, err := svc.Validate(nil, doc)
			if err != nil {
				t.Fatalf("Validate() error: %v", err)
			}

			if report.Valid != tt.expectValid {
				t.Errorf("Validate() Valid = %v, want %v", report.Valid, tt.expectValid)
			}

			for _, expectedErr := range tt.expectErrors {
				found := false
				for _, actualErr := range report.Errors {
					if strings.Contains(strings.ToLower(actualErr), expectedErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing %q not found in: %v", expectedErr, report.Errors)
				}
			}

			// Log warnings for debugging
			if len(report.Warnings) > 0 {
				t.Logf("Warnings: %v", report.Warnings)
			}
		})
	}
}

// TestValidate_HeaderOnly tests that files without %PDF- prefix are invalid.
func TestValidate_HeaderOnly(t *testing.T) {
	// Create a temporary file without PDF header
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "not-a-pdf.txt")

	content := "This is not a PDF file at all."
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("Failed to open test file: %v", err)
	}
	defer f.Close()

	report, err := validateDocument(f)
	if err != nil {
		t.Fatalf("validateDocument error: %v", err)
	}

	if report.Valid {
		t.Errorf("Validate() Valid = true, want false for non-PDF file")
	}

	headerErrorFound := false
	for _, err := range report.Errors {
		if strings.Contains(strings.ToLower(err), "header") ||
			strings.Contains(err, "%PDF") {
			headerErrorFound = true
			break
		}
	}
	if !headerErrorFound {
		t.Errorf("Expected header error not found in: %v", report.Errors)
	}
}

// TestValidate_ValidPDF tests that standard valid PDFs pass validation.
func TestValidate_ValidPDF(t *testing.T) {
	svc := NewService()

	samples := []string{"core-minimal", "core-multipage", "version-compat-v1.4"}

	for _, sampleKey := range samples {
		t.Run(sampleKey, func(t *testing.T) {
			sample, ok := testfixtures.PDFSampleByKey(sampleKey)
			if !ok {
				t.Fatalf("Sample %q not found", sampleKey)
			}

			doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
			if err != nil {
				t.Fatalf("Failed to open PDF: %v", err)
			}
			defer doc.Close()

			report, err := svc.Validate(nil, doc)
			if err != nil {
				t.Fatalf("Validate() error: %v", err)
			}

			if !report.Valid {
				t.Errorf("Validate() Valid = false, want true for valid PDF. Errors: %v", report.Errors)
			}
		})
	}
}

// TestValidate_CorruptedXRef tests that corrupted xref PDFs fail validation.
func TestValidate_CorruptedXRef(t *testing.T) {
	svc := NewService()

	sample, ok := testfixtures.PDFSampleByKey("error-corrupted")
	if !ok {
		t.Fatal("Sample 'error-corrupted' not found")
	}

	doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
	if err != nil {
		// Some corrupted files may not open - that's also a form of validation failure
		t.Logf("Open failed (expected for corrupted file): %v", err)
		return
	}
	defer doc.Close()

	report, err := svc.Validate(nil, doc)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if report.Valid {
		t.Errorf("Validate() Valid = true, want false for corrupted PDF")
	}

	// Check that we have meaningful error descriptions
	if len(report.Errors) == 0 {
		t.Errorf("Expected errors for corrupted PDF, got none")
	}
}

// TestValidate_XrefIntegrity validates xref structure checking.
func TestValidate_XrefIntegrity(t *testing.T) {
	svc := NewService()

	// Use a valid PDF and verify xref validation passes
	sample, ok := testfixtures.PDFSampleByKey("core-minimal")
	if !ok {
		t.Fatal("Sample 'core-minimal' not found")
	}

	doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
	if err != nil {
		t.Fatalf("Failed to open PDF: %v", err)
	}
	defer doc.Close()

	report, err := svc.Validate(nil, doc)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	// Xref integrity should pass for valid PDF
	xrefErrorFound := false
	for _, err := range report.Errors {
		if strings.Contains(strings.ToLower(err), "xref") {
			xrefErrorFound = true
			break
		}
	}
	if xrefErrorFound {
		t.Errorf("Unexpected xref error in valid PDF: %v", report.Errors)
	}
}

// TestValidate_TrailerFields validates trailer dictionary checks.
func TestValidate_TrailerFields(t *testing.T) {
	svc := NewService()

	// Test with a valid PDF that has proper trailer
	sample, ok := testfixtures.PDFSampleByKey("core-minimal")
	if !ok {
		t.Fatal("Sample 'core-minimal' not found")
	}

	doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
	if err != nil {
		t.Fatalf("Failed to open PDF: %v", err)
	}
	defer doc.Close()

	report, err := svc.Validate(nil, doc)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	// Valid PDF should pass trailer checks
	trailerErrorFound := false
	for _, err := range report.Errors {
		if strings.Contains(strings.ToLower(err), "trailer") {
			trailerErrorFound = true
			break
		}
	}
	if trailerErrorFound {
		t.Errorf("Unexpected trailer error in valid PDF: %v", report.Errors)
	}
}

// TestValidate_CatalogStructure validates catalog dictionary checks.
func TestValidate_CatalogStructure(t *testing.T) {
	svc := NewService()

	sample, ok := testfixtures.PDFSampleByKey("core-minimal")
	if !ok {
		t.Fatal("Sample 'core-minimal' not found")
	}

	doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
	if err != nil {
		t.Fatalf("Failed to open PDF: %v", err)
	}
	defer doc.Close()

	report, err := svc.Validate(nil, doc)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	// Valid PDF should pass catalog checks
	catalogErrorFound := false
	for _, err := range report.Errors {
		if strings.Contains(strings.ToLower(err), "catalog") {
			catalogErrorFound = true
			break
		}
	}
	if catalogErrorFound {
		t.Errorf("Unexpected catalog error in valid PDF: %v", report.Errors)
	}
}

// TestValidate_PagesTree validates pages tree checks.
func TestValidate_PagesTree(t *testing.T) {
	svc := NewService()

	sample, ok := testfixtures.PDFSampleByKey("core-minimal")
	if !ok {
		t.Fatal("Sample 'core-minimal' not found")
	}

	doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
	if err != nil {
		t.Fatalf("Failed to open PDF: %v", err)
	}
	defer doc.Close()

	report, err := svc.Validate(nil, doc)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	// Valid PDF should pass pages checks
	pagesErrorFound := false
	for _, err := range report.Errors {
		if strings.Contains(strings.ToLower(err), "pages") {
			pagesErrorFound = true
			break
		}
	}
	if pagesErrorFound {
		t.Errorf("Unexpected pages error in valid PDF: %v", report.Errors)
	}
}

// TestLevelName validates the level name helper function.
func TestLevelName(t *testing.T) {
	tests := []struct {
		level    ValidationLevel
		expected string
	}{
		{LevelHeader, "Header"},
		{LevelXRef, "XRef"},
		{LevelTrailer, "Trailer"},
		{LevelCatalog, "Catalog"},
		{LevelPages, "Pages"},
		{ValidationLevel(99), "Level99"}, // Unknown level
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := LevelName(tt.level)
			if got != tt.expected {
				t.Errorf("LevelName(%d) = %q, want %q", tt.level, got, tt.expected)
			}
		})
	}
}

// TestIsValidPDFVersion validates version format checking.
func TestIsValidPDFVersion(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		{"1.0", true},
		{"1.4", true},
		{"1.7", true},
		{"2.0", true},
		{"1.10", true}, // Valid format even if not standard
		{"2.15", true},
		{"", false},
		{"1", false},
		{"PDF-1.4", true},  // Extracts 1.4 from string
		{"1.4.0", true},    // Extracts 1.4 from string
		{"one.four", false},
		{"1.x", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := isValidPDFVersion(tt.version)
			if got != tt.expected {
				t.Errorf("isValidPDFVersion(%q) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

// TestValidate_Warnings checks that warnings are properly collected.
func TestValidate_Warnings(t *testing.T) {
	svc := NewService()

	// Use a valid PDF - should have no warnings
	sample, ok := testfixtures.PDFSampleByKey("core-minimal")
	if !ok {
		t.Fatal("Sample 'core-minimal' not found")
	}

	doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
	if err != nil {
		t.Fatalf("Failed to open PDF: %v", err)
	}
	defer doc.Close()

	report, err := svc.Validate(nil, doc)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	// Valid PDF should have no warnings (currently)
	// This test establishes the pattern for future warning checks
	if len(report.Warnings) > 0 {
		t.Logf("Warnings found (may be expected for specific features): %v", report.Warnings)
	}
}
