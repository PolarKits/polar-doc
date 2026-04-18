package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarKits/polardoc/internal/app"
	"github.com/PolarKits/polardoc/internal/doc"
)

func TestCLIE2EPDFReadWriteMatrix(t *testing.T) {
	resolver := app.NewPhase1Resolver()

	samples := []struct {
		name string
		path string
	}{
		{"pdf20-utf8", filepath.Join("..", "..", "..", "testdata", "pdf", "pdf20-utf8-test.pdf")},
		{"redhat-openshift", filepath.Join("..", "..", "..", "testdata", "pdf", "Red_Hat_OpenShift_Serverless-1.35-Serverless_Logic-en-US.pdf")},
		{"sample-local-pdf", filepath.Join("..", "..", "..", "testdata", "pdf", "sample-local-pdf.pdf")},
		{"testPDF-5x", filepath.Join("..", "..", "..", "testdata", "pdf", "testPDF_Version.5.x.pdf")},
		{"testPDF-8x", filepath.Join("..", "..", "..", "testdata", "pdf", "testPDF_Version.8.x.pdf")},
	}

	for _, tc := range samples {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := os.Stat(tc.path); os.IsNotExist(err) {
				t.Skipf("%s not found", tc.name)
			}

			t.Run("ReadFirstPageInfo", func(t *testing.T) {
				svc, ok := resolver.ByFormat(doc.FormatPDF)
				if !ok {
					t.Fatalf("no PDF service")
				}

				pdfDoc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: tc.path})
				if err != nil {
					t.Fatalf("Open failed: %v", err)
				}
				defer pdfDoc.Close()

				info, err := svc.FirstPageInfo(context.Background(), pdfDoc)
				if tc.name == "testPDF-8x" {
					if err == nil {
						t.Fatal("expected error for corrupted PDF")
					}
					t.Logf("correctly failed: %v", err)
				} else {
					if err != nil {
						t.Fatalf("FirstPageInfo failed: %v", err)
					}
					if info == nil {
						t.Fatal("info is nil")
					}
				}
			})

			t.Run("Copy", func(t *testing.T) {
				dst := filepath.Join(t.TempDir(), "copied.pdf")
				err := RunCopy(context.Background(), resolver, []string{tc.path, dst})
				if err != nil {
					t.Fatalf("Copy failed: %v", err)
				}

				if _, err := os.Stat(dst); err != nil {
					t.Fatalf("copied file not created: %v", err)
				}
			})
		})
	}
}
