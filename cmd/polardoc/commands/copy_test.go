package commands

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarKits/polardoc/internal/app"
)

func TestRunCopyPDFSuccess(t *testing.T) {
	src := filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.5.x.pdf")
	if _, err := os.Stat(src); os.IsNotExist(err) {
		t.Skip("testPDF_Version.5.x.pdf not found")
	}

	dst := filepath.Join(t.TempDir(), "copied.pdf")
	resolver := app.NewPhase1Resolver()

	if err := RunCopy(context.Background(), resolver, []string{src, dst}); err != nil {
		t.Fatalf("RunCopy failed: %v", err)
	}

	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("destination file not created: %v", err)
	}
}

func TestRunCopyOFDUnsupported(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	err := RunCopy(context.Background(), resolver, []string{"/tmp/sample.ofd", "/tmp/copy.ofd"})
	if err == nil {
		t.Fatal("RunCopy OFD should fail")
	}
	if err.Error() != "save not supported for OFD" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCopyWrongArgCount(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	err := RunCopy(context.Background(), resolver, []string{"/tmp/sample.pdf"})
	if err == nil {
		t.Fatal("RunCopy with 1 arg should fail")
	}
}
