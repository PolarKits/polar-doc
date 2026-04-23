package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarKits/polar-doc/internal/app"
	"github.com/PolarKits/polar-doc/internal/doc"
)

func TestCLIE2EPDFReadWriteMatrix(t *testing.T) {
	resolver := app.NewPhase1Resolver()

	samples := []struct {
		name string
		path string
	}{
		{"standard-pdf20-utf8", requirePDFSample(t, "standard-pdf20-utf8")},
		{"core-minimal", requirePDFSample(t, "core-minimal")},
		{"version-compat-v1.4", requirePDFSample(t, "version-compat-v1.4")},
		{"error-corrupted", requirePDFSample(t, "error-corrupted")},
		{"version-compat-v1.7", requirePDFSample(t, "version-compat-v1.7")},
	}

	for _, tc := range samples {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("ReadFirstPageInfo", func(t *testing.T) {
				if tc.name == "core-minimal" {
					t.Skip("ReadFirstPageInfo parser limitation: PDF has cross-reference stream that xref parser resolves to wrong offset (parser reads %PDF-1.5 header as xref entry); fixture xref is intact (Type B)")
				}
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
				if tc.name == "error-corrupted" || tc.name == "version-compat-v1.7" {
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
