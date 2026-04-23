package pdf

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
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

func TestCopyFileBytesMatchSource(t *testing.T) {
	src := requirePDFSample(t, "version-compat-v1.4")
	srcBytes, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("ReadFile source: %v", err)
	}

	dst := filepath.Join(t.TempDir(), "byte-match.pdf")
	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	dstBytes, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile destination: %v", err)
	}

	if !bytes.Equal(srcBytes, dstBytes) {
		t.Fatalf("destination bytes do not match source: src len=%d, dst len=%d", len(srcBytes), len(dstBytes))
	}
}

func TestCopyFileOverwriteExistingDest(t *testing.T) {
	src := requirePDFSample(t, "standard-pdf20-utf8")
	srcBytes, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("ReadFile source: %v", err)
	}

	dst := filepath.Join(t.TempDir(), "overwrite-target.pdf")
	if err := os.WriteFile(dst, []byte("different content"), 0644); err != nil {
		t.Fatalf("WriteFile initial destination: %v", err)
	}

	if err := CopyFile(src, dst); err != nil {
		t.Fatalf("CopyFile should succeed when destination exists: %v", err)
	}

	dstBytes, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile destination after copy: %v", err)
	}

	if !bytes.Equal(srcBytes, dstBytes) {
		t.Fatalf("destination was not overwritten correctly: expected %d bytes, got %d", len(srcBytes), len(dstBytes))
	}
}

func TestCopyFileSourceNotFound(t *testing.T) {
	nonExistent := filepath.Join(t.TempDir(), "nonexistent-src-"+t.Name()+".pdf")
	err := CopyFile(nonExistent, filepath.Join(t.TempDir(), "dst.pdf"))
	if err == nil {
		t.Fatal("CopyFile should fail when source does not exist")
	}
	t.Logf("CopyFile with nonexistent source returns error: %v", err)
}

func TestCopyFileDestDirMissing(t *testing.T) {
	src := requirePDFSample(t, "core-multipage")
	dst := filepath.Join(t.TempDir(), "nonexistent-dir-"+t.Name(), "output.pdf")
	err := CopyFile(src, dst)
	if err == nil {
		t.Fatal("CopyFile should fail when destination directory does not exist")
	}
	t.Logf("CopyFile with missing dest dir returns error: %v", err)
}

func TestRewriteFileCoreMinimal(t *testing.T) {
	src := requirePDFSample(t, "core-minimal")
	dst := filepath.Join(t.TempDir(), "rewritten.pdf")
	if err := RewriteFile(src, dst); err != nil {
		t.Fatalf("RewriteFile failed: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: dst})
	if err != nil {
		t.Fatalf("open rewritten PDF: %v", err)
	}
	defer d.Close()

	info, err := svc.Info(context.Background(), d)
	if err != nil {
		t.Fatalf("Info on rewritten PDF: %v", err)
	}
	if info.Format != "pdf" {
		t.Fatalf("format = %q, want pdf", info.Format)
	}
	if info.SizeBytes == 0 {
		t.Fatal("rewritten PDF has zero size")
	}
}

func TestRewriteFilePreservesInfo(t *testing.T) {
	src := requirePDFSample(t, "version-compat-v1.4")
	dst := filepath.Join(t.TempDir(), "rewritten.pdf")
	if err := RewriteFile(src, dst); err != nil {
		t.Fatalf("RewriteFile failed: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: dst})
	if err != nil {
		t.Fatalf("open rewritten PDF: %v", err)
	}
	defer d.Close()

	info, err := svc.Info(context.Background(), d)
	if err != nil {
		t.Fatalf("Info on rewritten PDF: %v", err)
	}
	if info.Title == "" {
		t.Fatal("rewritten PDF has empty Title; metadata was lost")
	}
	t.Logf("rewritten PDF title: %q", info.Title)
}

func TestRewriteFilePDF20UTF8(t *testing.T) {
	src := requirePDFSample(t, "standard-pdf20-utf8")
	dst := filepath.Join(t.TempDir(), "rewritten.pdf")
	if err := RewriteFile(src, dst); err != nil {
		t.Fatalf("RewriteFile failed: %v", err)
	}

	svc := NewService()
	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: dst})
	if err != nil {
		t.Fatalf("open rewritten PDF: %v", err)
	}
	defer d.Close()

	_, err = svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		t.Fatalf("FirstPageInfo on rewritten PDF: %v", err)
	}
}

func TestRewriteFileEmptyPath(t *testing.T) {
	if err := RewriteFile("", "/tmp/dst.pdf"); err == nil {
		t.Fatal("RewriteFile with empty source should fail")
	}
	if err := RewriteFile("/tmp/src.pdf", ""); err == nil {
		t.Fatal("RewriteFile with empty destination should fail")
	}
}

func TestRewriteFileIsSingleRevision(t *testing.T) {
	// An incremental PDF has multiple startxref markers. After RewriteFile,
	// the output should be a clean single-revision PDF with exactly one startxref.
	src := requirePDFSample(t, "version-compat-v1.4")
	dst := filepath.Join(t.TempDir(), "rewritten.pdf")
	if err := RewriteFile(src, dst); err != nil {
		t.Fatalf("RewriteFile failed: %v", err)
	}

	dstBytes, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	count := bytes.Count(dstBytes, []byte("startxref"))
	if count != 1 {
		t.Fatalf("expected exactly 1 startxref in rewritten PDF, got %d", count)
	}
}
