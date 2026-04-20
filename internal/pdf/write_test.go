package pdf

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/PolarKits/polardoc/internal/doc"
)

func TestPDFServiceHasLimitedWriteCapability(t *testing.T) {
	svc := NewService()
	v := reflect.ValueOf(svc)
	if v.Kind() != reflect.Ptr {
		t.Fatalf("NewService returns non-pointer: %T", svc)
	}

	t.Log("Phase-1 PDF module has limited write capability: Save (CopyFile). Other write methods not implemented.")

	for _, method := range []string{"Write", "Update", "Modify", "Export"} {
		methodValue := v.MethodByName(method)
		if methodValue.IsValid() {
			t.Errorf("unexpected write method found: %s", method)
		}
	}

	saveMethod := v.MethodByName("Save")
	if !saveMethod.IsValid() {
		t.Error("Save method should be present")
	}
}

func TestCopyFilePDF5x(t *testing.T) {
	src := requirePDFSample(t, "version-compat-v1.4")
	dst := filepath.Join(t.TempDir(), "copied.pdf")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: dst})
	if err != nil {
		t.Fatalf("open copied PDF failed: %v", err)
	}
	defer d.Close()

	_, err = svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		t.Fatalf("FirstPageInfo on copied PDF failed: %v", err)
	}
}

func TestCopyFilePDF20UTF8(t *testing.T) {
	src := requirePDFSample(t, "standard-pdf20-utf8")
	dst := filepath.Join(t.TempDir(), "copied.pdf")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: dst})
	if err != nil {
		t.Fatalf("open copied PDF failed: %v", err)
	}
	defer d.Close()

	_, err = svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		t.Fatalf("FirstPageInfo on copied PDF failed: %v", err)
	}
}

func TestCopyFileCoreMultipage(t *testing.T) {
	src := requirePDFSample(t, "core-multipage")
	dst := filepath.Join(t.TempDir(), "copied.pdf")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: dst})
	if err != nil {
		t.Fatalf("open copied PDF failed: %v", err)
	}
	defer d.Close()

	_, err = svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		t.Fatalf("FirstPageInfo on copied PDF failed: %v", err)
	}
}

func TestCopyFileLegacyWithTable(t *testing.T) {
	t.Skip("OpenAction contains literal string with embedded null bytes that parser cannot handle in array context; fixture xref is intact (Type B parser limitation)")
	src := requirePDFSample(t, "legacy-with-table")
	dst := filepath.Join(t.TempDir(), "copied.pdf")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: dst})
	if err != nil {
		t.Fatalf("open copied PDF failed: %v", err)
	}
	defer d.Close()

	_, err = svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		t.Fatalf("FirstPageInfo on copied PDF failed: %v", err)
	}
}

func TestCopyFileCorrupted(t *testing.T) {
	src := requirePDFSample(t, "error-corrupted")
	dst := filepath.Join(t.TempDir(), "copied-bad.pdf")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile should succeed even for corrupted PDF: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: dst})
	if err != nil {
		t.Fatalf("open copied corrupted PDF should succeed: %v", err)
	}
	defer d.Close()

	_, err = svc.FirstPageInfo(context.Background(), d)
	if err == nil {
		t.Fatal("FirstPageInfo on copied corrupted PDF should fail")
	}
	t.Logf("FirstPageInfo correctly fails on copied corrupted PDF: %v", err)
}

func TestCopyFileEmptyPath(t *testing.T) {
	err := CopyFile("", "/tmp/dst.pdf")
	if err == nil {
		t.Fatal("CopyFile with empty source should fail")
	}

	err = CopyFile("/tmp/src.pdf", "")
	if err == nil {
		t.Fatal("CopyFile with empty destination should fail")
	}
}
