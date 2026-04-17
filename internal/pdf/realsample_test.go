package pdf

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarKits/polardoc/internal/doc"
)

type PDFSampleResult struct {
	filename         string
	openInfoOK       bool
	openInfoErr      string
	validateOK       bool
	validateErr      string
	firstPageInfoOK  bool
	firstPageInfoErr string
}

var realPDFSampleResults []PDFSampleResult

// TestPDFRealSamples exercises Open+Info, Validate, and ReadFirstPageInfo
// on each real PDF file in testdata/pdf and records the results for the compatibility matrix.
func TestPDFRealSamples(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata", "pdf")

	tests := []struct {
		filename string
	}{
		{"pdf20-utf8-test.pdf"},
		{"Red_Hat_OpenShift_Serverless-1.35-Serverless_Logic-en-US.pdf"},
		{"sample-local-pdf.pdf"},
		{"testPDF_Version.5.x.pdf"},
		{"testPDF_Version.8.x.pdf"},
	}

	realPDFSampleResults = nil

	for _, tt := range tests {
		result := PDFSampleResult{filename: tt.filename}
		path := filepath.Join(testdataDir, tt.filename)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Skipf("testdata file %s not found", tt.filename)
			continue
		}

		svc := NewService()

		d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: path})
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

			d.Close()
		}

		f, err := os.Open(path)
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
			f.Close()
		}

		realPDFSampleResults = append(realPDFSampleResults, result)
	}

	if len(realPDFSampleResults) == 0 {
		t.Fatal("no PDF samples tested")
	}

	t.Log("\n=== Real PDF Sample Compatibility Matrix ===")
	for _, r := range realPDFSampleResults {
		t.Logf("File: %s", r.filename)
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

func TestPDFKnownBad_Version8x(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.8.x.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testPDF_Version.8.x.pdf not found")
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo should fail for testPDF_Version.8.x.pdf (XRef corrupted: object 14 referenced but marked as free)")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "object 14") || !strings.Contains(errMsg, "not found in xref") {
		t.Fatalf("expected error about object 14 not found in xref, got: %s", errMsg)
	}

	t.Logf("testPDF_Version.8.x.pdf: ReadFirstPageInfo correctly fails with identified XRef corruption: %s", errMsg)
}

func TestPDFRecovery_Version8x(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.8.x.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testPDF_Version.8.x.pdf not found")
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open: %v", err)
	}
	defer f.Close()

	_, err = ReadFirstPageInfo(f)
	if err == nil {
		t.Fatal("ReadFirstPageInfo should still fail after recovery attempt: object 14 content does not exist in file body")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "object 14") {
		t.Fatalf("expected error about object 14, got: %s", errMsg)
	}

	t.Logf("testPDF_Version.8.x.pdf: recovery path attempted but object 14 content not in file body: %s", errMsg)
}
