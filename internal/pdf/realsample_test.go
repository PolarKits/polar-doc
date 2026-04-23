package pdf

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
	fixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

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
