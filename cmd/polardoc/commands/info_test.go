package commands

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarKits/polardoc/internal/app"
)

func TestRunInfoPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.4\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{path}); err != nil {
			t.Fatalf("run info PDF: %v", err)
		}
	})

	mustContain(t, output, "format: pdf")
	mustContain(t, output, "path: "+path)
	mustContain(t, output, "size_bytes: 9")
	mustContain(t, output, "declared_version: 1.4")
}

func TestRunInfoOFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{path}); err != nil {
			t.Fatalf("run info OFD: %v", err)
		}
	})

	mustContain(t, output, "format: ofd")
	mustContain(t, output, "path: "+path)
	mustContain(t, output, "size_bytes: ")
	if strings.Contains(output, "declared_version:") {
		t.Fatalf("unexpected declared_version output: %q", output)
	}
}

func TestRunInfoJSONPDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	content := []byte("%PDF-1.4\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample PDF: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{"--json", path}); err != nil {
			t.Fatalf("run info PDF JSON: %v", err)
		}
	})

	var got struct {
		Format          string   `json:"format"`
		Path            string   `json:"path"`
		SizeBytes       int64    `json:"size_bytes"`
		DeclaredVersion string   `json:"declared_version"`
		PageCount       int      `json:"page_count"`
		FileIdentifiers []string `json:"file_identifiers"`
		Title           string   `json:"title"`
		Author          string   `json:"author"`
		Creator         string   `json:"creator"`
		Producer        string   `json:"producer"`
	}
	mustUnmarshalJSON(t, output, &got)

	if got.Format != "pdf" {
		t.Fatalf("format = %q, want %q", got.Format, "pdf")
	}
	if got.Path != path {
		t.Fatalf("path = %q, want %q", got.Path, path)
	}
	if got.SizeBytes != int64(len(content)) {
		t.Fatalf("size_bytes = %d, want %d", got.SizeBytes, len(content))
	}
	if got.DeclaredVersion != "1.4" {
		t.Fatalf("declared_version = %q, want %q", got.DeclaredVersion, "1.4")
	}
	if got.PageCount != 0 {
		t.Fatalf("page_count = %d, want 0", got.PageCount)
	}
	if got.FileIdentifiers != nil {
		t.Fatalf("file_identifiers = %v, want nil", got.FileIdentifiers)
	}
	if got.Title != "" {
		t.Fatalf("title = %q, want empty", got.Title)
	}
	if got.Author != "" {
		t.Fatalf("author = %q, want empty", got.Author)
	}
	if got.Creator != "" {
		t.Fatalf("creator = %q, want empty", got.Creator)
	}
	if got.Producer != "" {
		t.Fatalf("producer = %q, want empty", got.Producer)
	}
}

func TestRunInfoJSONOFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{"--json", path}); err != nil {
			t.Fatalf("run info OFD JSON: %v", err)
		}
	})

	var got map[string]any
	mustUnmarshalJSON(t, output, &got)

	if got["format"] != "ofd" {
		t.Fatalf("format = %v, want %q", got["format"], "ofd")
	}
	if got["path"] != path {
		t.Fatalf("path = %v, want %q", got["path"], path)
	}
	if _, ok := got["declared_version"]; ok {
		t.Fatalf("unexpected declared_version in output: %q", output)
	}
}

func captureStdout(t *testing.T, run func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}

	os.Stdout = w
	run()
	_ = w.Close()
	os.Stdout = oldStdout

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return string(data)
}

func mustContain(t *testing.T, output, expected string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Fatalf("output = %q, want contains %q", output, expected)
	}
}

func mustUnmarshalJSON(t *testing.T, output string, dst any) {
	t.Helper()
	if err := json.Unmarshal([]byte(output), dst); err != nil {
		t.Fatalf("unmarshal JSON output %q: %v", output, err)
	}
}

func buildOFDPackage(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %q: %v", name, err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatalf("write zip entry %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestRunInfoPagePDF(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "pdf", "testPDF_Version.5.x.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testPDF_Version.5.x.pdf not found")
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{"--page", path}); err != nil {
			t.Fatalf("run info --page: %v", err)
		}
	})

	mustContain(t, output, "path: "+path)
	mustContain(t, output, "pages_ref:")
	mustContain(t, output, "page_ref:")
	mustContain(t, output, "parent:")
	mustContain(t, output, "media_box:")
	mustContain(t, output, "resources:")
}

func TestRunInfoPageJSON(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "pdf", "testPDF_Version.5.x.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testPDF_Version.5.x.pdf not found")
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{"--json", "--page", path}); err != nil {
			t.Fatalf("run info --json --page: %v", err)
		}
	})

	var got pageInfoOutput
	mustUnmarshalJSON(t, output, &got)

	if got.Path != path {
		t.Fatalf("path = %q, want %q", got.Path, path)
	}
	if got.PagesRef.ObjNum == 0 {
		t.Fatalf("pages_ref obj_num is zero")
	}
	if got.PageRef.ObjNum == 0 {
		t.Fatalf("page_ref obj_num is zero")
	}
	if got.Parent.ObjNum == 0 {
		t.Fatalf("parent obj_num is zero")
	}
	if len(got.MediaBox) == 0 {
		t.Fatalf("media_box is empty")
	}
	if got.Resources.ObjNum == 0 && len(got.Contents) == 0 {
		t.Fatalf("both resources obj_num and contents are zero (inline resources not supported in JSON output)")
	}
}

func TestRunInfoPageKnownBad(t *testing.T) {
	path := filepath.Join("..", "..", "..", "testdata", "pdf", "testPDF_Version.8.x.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testPDF_Version.8.x.pdf not found")
	}

	resolver := app.NewPhase1Resolver()
	err := RunInfo(context.Background(), resolver, []string{"--page", path})
	if err == nil {
		t.Fatal("run info --page should fail for known-bad PDF")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "object 14") {
		t.Fatalf("expected error about object 14, got: %s", errMsg)
	}

	t.Logf("info --page correctly fails for known-bad PDF: %s", errMsg)
}

func TestRunInfoPageOFDUnsupported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            "<ofd/>",
		"Doc_0/Document.xml": "<document/>",
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	err := RunInfo(context.Background(), resolver, []string{"--page", path})
	if err == nil {
		t.Fatal("run info --page OFD should fail")
	}

	if !strings.Contains(err.Error(), "--page is only supported for PDF") {
		t.Fatalf("expected --page is only supported for PDF, got: %s", err.Error())
	}
}
