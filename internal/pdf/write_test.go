package pdf

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/PolarKits/polardoc/internal/doc"
)

func TestPDFServiceHasNoWriteCapability(t *testing.T) {
	svc := NewService()
	v := reflect.ValueOf(svc)
	if v.Kind() != reflect.Ptr {
		t.Fatalf("NewService returns non-pointer: %T", svc)
	}

	t.Log("Phase-1 PDF module has read-only capabilities. Write capabilities are not implemented.")

	for _, method := range []string{"Write", "Save", "Update", "Modify", "Export"} {
		methodValue := v.MethodByName(method)
		if methodValue.IsValid() {
			t.Errorf("unexpected write method found: %s", method)
		}
	}
}

func TestCopyFilePDF5x(t *testing.T) {
	src := filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.5.x.pdf")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Skip("testPDF_Version.5.x.pdf not found")
	}

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
	src := filepath.Join("..", "..", "testdata", "pdf", "pdf20-utf8-test.pdf")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Skip("pdf20-utf8-test.pdf not found")
	}

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

func TestCopyFileRedHatOpenShift(t *testing.T) {
	src := filepath.Join("..", "..", "testdata", "pdf", "Red_Hat_OpenShift_Serverless-1.35-Serverless_Logic-en-US.pdf")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Skip("Red_Hat_OpenShift_Serverless-1.35-Serverless_Logic-en-US.pdf not found")
	}

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

func TestCopyFileSampleLocal(t *testing.T) {
	src := filepath.Join("..", "..", "testdata", "pdf", "sample-local-pdf.pdf")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Skip("sample-local-pdf.pdf not found")
	}

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
	src := filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.8.x.pdf")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Skip("testPDF_Version.8.x.pdf not found")
	}

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
