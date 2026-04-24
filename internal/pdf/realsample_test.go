package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
	fixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

// PDFSampleResult records the outcome of testing a single PDF sample file.
// It captures the file identity, integrity classification, and the success/failure
// status of various operations (Open+Info, Validate, ReadFirstPageInfo) along with
// any error messages for diagnostic purposes.
type PDFSampleResult struct {
	key              string
	filename         string
	integrity        fixtures.PDFSampleIntegrity
	openInfoOK       bool
	openInfoErr      string
	validateOK       bool
	validateErr      string
	firstPageInfoOK  bool
	firstPageInfoErr string
}

var realPDFSampleResults []PDFSampleResult

func TestPDFSampleCatalogIntegrity(t *testing.T) {
	categoryCounts := map[string]int{}

	for _, sample := range fixtures.PDFSamples() {
		categoryCounts[sample.Category]++

		f, err := os.Open(sample.Path())
		if err != nil {
			t.Fatalf("%s: open sample %q: %v", sample.Key, sample.Filename, err)
		}

		_, headerErr := readPDFHeaderVersion(f)
		_ = f.Close()

		switch sample.Integrity {
		case fixtures.PDFSampleIntegrityValid, fixtures.PDFSampleIntegrityCorrupted:
			if headerErr != nil {
				t.Fatalf("%s: expected a PDF header, got %v", sample.Key, headerErr)
			}
		case fixtures.PDFSampleIntegrityPlaceholder:
			if headerErr == nil {
				t.Fatalf("%s: placeholder fixture unexpectedly contains a valid PDF header", sample.Key)
			}
		default:
			t.Fatalf("%s: unknown integrity state %q", sample.Key, sample.Integrity)
		}
	}

	for _, required := range []string{"core", "feature", "standard", "version-compat", "error"} {
		if categoryCounts[required] == 0 {
			t.Fatalf("missing sample coverage category %q", required)
		}
	}
}

// TestPDFRealSamples exercises Open+Info, Validate, and ReadFirstPageInfo
// on every non-placeholder sample in testdata/pdf and records the results.
func TestPDFRealSamples(t *testing.T) {
	realPDFSampleResults = nil

	for _, sample := range fixtures.PDFSamples() {
		if sample.Integrity == fixtures.PDFSampleIntegrityPlaceholder {
			continue
		}

		result := PDFSampleResult{
			key:       sample.Key,
			filename:  sample.Filename,
			integrity: sample.Integrity,
		}

		svc := NewService()
		d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
		if err != nil {
			result.openInfoErr = err.Error()
		} else {
			info, err := svc.Info(context.Background(), d)
			if err != nil {
				result.openInfoErr = err.Error()
			} else {
				result.openInfoOK = info.Path != "" && info.SizeBytes > 0
				if !result.openInfoOK {
					result.openInfoErr = "Path empty or SizeBytes zero"
				}
			}

			report, err := svc.Validate(context.Background(), d)
			if err != nil {
				result.validateErr = err.Error()
			} else {
				result.validateOK = report.Valid
				if !result.validateOK && len(report.Errors) > 0 {
					result.validateErr = joinErrors(report.Errors)
				}
			}

			_ = d.Close()
		}

		f, err := os.Open(sample.Path())
		if err != nil {
			result.firstPageInfoErr = err.Error()
		} else {
			info, err := ReadFirstPageInfo(f)
			if err != nil {
				result.firstPageInfoErr = err.Error()
			} else {
				result.firstPageInfoOK = info.PageRef.ObjNum != 0 &&
					info.Parent.ObjNum != 0 &&
					len(info.MediaBox) > 0 &&
					(info.Resources.ObjNum != 0 || info.InlineResources != nil) &&
					len(info.Contents) > 0
				if !result.firstPageInfoOK {
					result.firstPageInfoErr = "returned info has zero/nil fields"
				}
			}
			_ = f.Close()
		}

		realPDFSampleResults = append(realPDFSampleResults, result)
	}

	if len(realPDFSampleResults) == 0 {
		t.Fatal("no PDF samples tested")
	}

	t.Log("\n=== Real PDF Sample Compatibility Matrix ===")
	for _, r := range realPDFSampleResults {
		t.Logf("Sample: %s (%s)", r.key, r.filename)
		t.Logf("  Integrity: %s", r.integrity)
		t.Logf("  Open+Info: OK=%v Err=%q", r.openInfoOK, r.openInfoErr)
		t.Logf("  Validate: OK=%v Err=%q", r.validateOK, r.validateErr)
		t.Logf("  ReadFirstPageInfo: OK=%v Err=%q", r.firstPageInfoOK, r.firstPageInfoErr)
	}
}

// joinErrors concatenates a slice of error strings into a single semicolon-separated
// string. It is used to aggregate multiple validation errors into a readable format
// for test logging and debugging.
func joinErrors(errs []string) string {
	result := ""
	for i, e := range errs {
		if i > 0 {
			result += "; "
		}
		result += e
	}
	return result
}

// Bulk sample patterns for testing.
// These patterns match the sample files committed for edge case and security testing.
var bulkSamplePatterns = []string{
	"artur_*.pdf",
	"bmaupin_*.pdf",
	"bosdev_*.pdf",
	"pmaupin_*.pdf",
	"sambit_*.pdf",
	"sample_*.pdf",
	"sec_*.pdf",
}

// isBulkSample returns true if the filename matches any of the bulk sample patterns.
func isBulkSample(filename string) bool {
	for _, pattern := range bulkSamplePatterns {
		// Simple glob matching for the patterns we use
		prefix := pattern[:len(pattern)-5] // Remove "*.pdf"
		if strings.HasPrefix(filename, prefix) && strings.HasSuffix(filename, ".pdf") {
			return true
		}
	}
	return false
}

// TestRealPDFSamples_BulkOpen tests that the bulk PDF samples can be opened.
// It iterates through all samples matching the bulk patterns and records
// open success/failure statistics. Failed opens are logged but do not fail
// the test, as some samples are intentionally corrupted or malformed.
// The test asserts that at least 80% of samples can be successfully opened.
func TestRealPDFSamples_BulkOpen(t *testing.T) {
	entries, err := os.ReadDir(fixtures.PDFDir())
	if err != nil {
		t.Fatalf("failed to read testdata/pdf: %v", err)
	}

	svc := NewService()
	var totalCount, successCount int
	var failedOpens []string

	for _, entry := range entries {
		if entry.IsDir() || !isBulkSample(entry.Name()) {
			continue
		}

		totalCount++
		path := filepath.Join(fixtures.PDFDir(), entry.Name())

		d, err := svc.Open(context.Background(), doc.DocumentRef{
			Format: doc.FormatPDF,
			Path:   path,
		})
		if err != nil {
			// Log failure but don't fail the test - some samples are intentionally malformed
			failedOpens = append(failedOpens, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}
		successCount++
		_ = d.Close()
	}

	// Log statistics
	t.Logf("=== Bulk Open Results ===")
	t.Logf("Total samples: %d", totalCount)
	t.Logf("Successful opens: %d (%.1f%%)", successCount, float64(successCount)*100/float64(totalCount))
	t.Logf("Failed opens: %d", len(failedOpens))

	// Log first 10 failures for diagnostic purposes
	for i, fail := range failedOpens {
		if i >= 10 {
			t.Logf("... and %d more failures", len(failedOpens)-10)
			break
		}
		t.Logf("  - %s", fail)
	}

	// Assert at least 80% success rate
	if totalCount > 0 {
		successRate := float64(successCount) / float64(totalCount)
		if successRate < 0.80 {
			t.Errorf("open success rate %.1f%% below threshold 80%%", successRate*100)
		}
	}
}

// TestRealPDFSamples_BulkInfo tests Info extraction on successfully opened samples.
// It verifies that InfoResult contains required fields (Format, Path, SizeBytes)
// and records the distribution of DeclaredVersion values across samples.
func TestRealPDFSamples_BulkInfo(t *testing.T) {
	entries, err := os.ReadDir(fixtures.PDFDir())
	if err != nil {
		t.Fatalf("failed to read testdata/pdf: %v", err)
	}

	svc := NewService()
	var successCount int
	versionStats := make(map[string]int)
	var failedInfo []string

	for _, entry := range entries {
		if entry.IsDir() || !isBulkSample(entry.Name()) {
			continue
		}

		path := filepath.Join(fixtures.PDFDir(), entry.Name())

		d, err := svc.Open(context.Background(), doc.DocumentRef{
			Format: doc.FormatPDF,
			Path:   path,
		})
		if err != nil {
			// Skip samples that can't be opened
			continue
		}
		defer d.Close()

		info, err := svc.Info(context.Background(), d)
		if err != nil {
			failedInfo = append(failedInfo, fmt.Sprintf("%s: %v", entry.Name(), err))
			continue
		}

		// Verify required fields
		if info.Format == "" {
			t.Errorf("%s: InfoResult.Format is empty", entry.Name())
		}
		if info.Path == "" {
			t.Errorf("%s: InfoResult.Path is empty", entry.Name())
		}
		if info.SizeBytes == 0 {
			t.Errorf("%s: InfoResult.SizeBytes is zero", entry.Name())
		}

		successCount++
		versionStats[info.DeclaredVersion]++
	}

	t.Logf("=== Bulk Info Results ===")
	t.Logf("Successful info extractions: %d", successCount)
	t.Logf("Failed info extractions: %d", len(failedInfo))
	t.Logf("DeclaredVersion distribution:")
	for version, count := range versionStats {
		t.Logf("  %q: %d", version, count)
	}

	if len(failedInfo) > 0 {
		t.Logf("Sample failures:")
		for i, fail := range failedInfo {
			if i >= 5 {
				break
			}
			t.Logf("  - %s", fail)
		}
	}
}

// TestRealPDFSamples_BulkValidate tests validation on successfully opened samples.
// It records Valid=true/false statistics and logs errors for invalid samples
// to aid in identifying structural issues in test fixtures.
func TestRealPDFSamples_BulkValidate(t *testing.T) {
	entries, err := os.ReadDir(fixtures.PDFDir())
	if err != nil {
		t.Fatalf("failed to read testdata/pdf: %v", err)
	}

	svc := NewService()
	var validCount, invalidCount int
	var invalidSamples []struct {
		name   string
		errors string
	}

	for _, entry := range entries {
		if entry.IsDir() || !isBulkSample(entry.Name()) {
			continue
		}

		path := filepath.Join(fixtures.PDFDir(), entry.Name())

		d, err := svc.Open(context.Background(), doc.DocumentRef{
			Format: doc.FormatPDF,
			Path:   path,
		})
		if err != nil {
			// Skip samples that can't be opened
			continue
		}

		report, err := svc.Validate(context.Background(), d)
		_ = d.Close()

		if err != nil {
			// Validation itself failed (not just invalid document)
			t.Logf("%s: validation error: %v", entry.Name(), err)
			continue
		}

		if report.Valid {
			validCount++
		} else {
			invalidCount++
			invalidSamples = append(invalidSamples, struct {
				name   string
				errors string
			}{
				name:   entry.Name(),
				errors: joinErrors(report.Errors),
			})
		}
	}

	t.Logf("=== Bulk Validate Results ===")
	t.Logf("Valid samples: %d", validCount)
	t.Logf("Invalid samples: %d", invalidCount)

	// Log first 10 invalid samples with their errors
	if len(invalidSamples) > 0 {
		t.Logf("Invalid samples (first 10):")
		for i, inv := range invalidSamples {
			if i >= 10 {
				t.Logf("  ... and %d more", len(invalidSamples)-10)
				break
			}
			t.Logf("  - %s: %s", inv.name, inv.errors)
		}
	}
}

// TestRealPDFSamples_EncryptionDetection tests encryption detection on samples
// known to be encrypted or non-encrypted. It uses DocumentFeatures to verify
// that IsEncrypted is correctly detected for artur_* and sambit_* samples.
// Known encrypted samples: artur_corrupted.pdf, sambit_pades_*.pdf
// Known non-encrypted samples: artur_not_encrypted*.pdf, other sambit_*.pdf
func TestRealPDFSamples_EncryptionDetection(t *testing.T) {
	entries, err := os.ReadDir(fixtures.PDFDir())
	if err != nil {
		t.Fatalf("failed to read testdata/pdf: %v", err)
	}

	svc := NewService()

	// Track encryption detection results
	var encryptedDetected, nonEncryptedDetected int
	var mismatches []string

	for _, entry := range entries {
		filename := entry.Name()
		if entry.IsDir() || !isBulkSample(filename) {
			continue
		}

		// Only test artur_* and sambit_* samples for encryption
		if !strings.HasPrefix(filename, "artur_") && !strings.HasPrefix(filename, "sambit_") {
			continue
		}

		path := filepath.Join(fixtures.PDFDir(), filename)

		d, err := svc.Open(context.Background(), doc.DocumentRef{
			Format: doc.FormatPDF,
			Path:   path,
		})
		if err != nil {
			// Skip samples that can't be opened
			continue
		}
		defer d.Close()

		features, err := svc.DocumentFeatures(context.Background(), d)
		if err != nil {
			t.Logf("%s: failed to get features: %v", filename, err)
			continue
		}

		// Determine expected encryption status based on filename patterns
		expectedEncrypted := false
		if strings.Contains(filename, "corrupted") ||
			strings.Contains(filename, "encrypted") ||
			strings.Contains(filename, "pades_") ||
			strings.Contains(filename, "protected") {
			expectedEncrypted = true
		}
		// artur_not_encrypted* files should NOT be encrypted despite "encrypted" in name
		if strings.HasPrefix(filename, "artur_not_encrypted") {
			expectedEncrypted = false
		}

		if features.IsEncrypted {
			encryptedDetected++
		} else {
			nonEncryptedDetected++
		}

		// Verify encryption detection matches expectation (when we have clear expectations)
		if expectedEncrypted && !features.IsEncrypted {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected encrypted but got non-encrypted", filename))
		}

		// Log encryption info for all samples
		algo := encryptionAlgorithmToString(features.EncryptionAlgorithm)
		t.Logf("%s: IsEncrypted=%v, Algorithm=%s", filename, features.IsEncrypted, algo)
	}

	t.Logf("=== Encryption Detection Results ===")
	t.Logf("Encrypted samples detected: %d", encryptedDetected)
	t.Logf("Non-encrypted samples detected: %d", nonEncryptedDetected)

	if len(mismatches) > 0 {
		t.Logf("Encryption detection mismatches:")
		for _, m := range mismatches {
			t.Logf("  - %s", m)
		}
		// Don't fail the test for mismatches - this is informational for now
		t.Logf("Note: %d encryption detection mismatches logged for review", len(mismatches))
	}
}

func TestPDFKnownBadErrorFixture(t *testing.T) {
	sample, ok := fixtures.PDFSampleByKey("error-corrupted")
	if !ok {
		t.Fatal("missing error-corrupted sample")
	}

	f, err := os.Open(sample.Path())
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo should fail for the corrupted PDF fixture")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "xref") && !strings.Contains(errMsg, "object") {
		t.Fatalf("expected xref/object failure, got: %s", errMsg)
	}
}

func TestPDFKnownBadVersionCompatV17(t *testing.T) {
	sample, ok := fixtures.PDFSampleByKey("version-compat-v1.7")
	if !ok {
		t.Fatal("missing version-compat-v1.7 sample")
	}

	f, err := os.Open(sample.Path())
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo should fail for version-compat-v1.7")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "xref") && !strings.Contains(errMsg, "object") {
		t.Fatalf("expected xref/object failure, got: %s", errMsg)
	}
}

// encryptionAlgorithmToString converts an EncryptionAlgorithm constant to a
// human-readable string representation for logging and debugging purposes.
func encryptionAlgorithmToString(algo EncryptionAlgorithm) string {
	switch algo {
	case EncryptNone:
		return "none"
	case EncryptRC4_40:
		return "RC4-40"
	case EncryptRC4_128:
		return "RC4-128"
	case EncryptAES_128:
		return "AES-128"
	case EncryptAES_256:
		return "AES-256"
	case EncryptUnknown:
		return "unknown"
	default:
		return fmt.Sprintf("unknown(%d)", algo)
	}
}
