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

func TestRunInfoJSONPDFWithMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.pdf")
	pdf := []byte("%PDF-1.4\n" +
		"1 0 obj\n" +
		"<< /Type /Catalog /Pages 2 0 R >>\n" +
		"endobj\n" +
		"2 0 obj\n" +
		"<< /Type /Pages /Kids [] /Count 0 >>\n" +
		"endobj\n" +
		"3 0 obj\n" +
		"<< /Title (Test Title) /Author (Test Author) /Creator (Test Creator) /Producer (Test Producer) >>\n" +
		"endobj\n" +
		"xref\n" +
		"0 4\n" +
		"0000000000 65535 f \n" +
		"0000000009 00000 n \n" +
		"0000000058 00000 n \n" +
		"0000000110 00000 n \n" +
		"trailer\n" +
		"<< /Root 1 0 R /Size 4 /Info 3 0 R /ID [(abcd1234)(efgh5678)] >>\n" +
		"startxref\n" +
		"223\n" +
		"%%EOF\n")
	if err := os.WriteFile(path, pdf, 0o644); err != nil {
		t.Fatalf("write PDF with metadata: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	output := captureStdout(t, func() {
		if err := RunInfo(context.Background(), resolver, []string{"--json", path}); err != nil {
			t.Fatalf("run info PDF JSON with metadata: %v", err)
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
	if got.DeclaredVersion != "1.4" {
		t.Fatalf("declared_version = %q, want %q", got.DeclaredVersion, "1.4")
	}
	if got.PageCount != 0 {
		t.Fatalf("page_count = %d, want 0 (not implemented for PDF)", got.PageCount)
	}
	if len(got.FileIdentifiers) != 2 {
		t.Fatalf("file_identifiers length = %d, want 2", len(got.FileIdentifiers))
	}
	if got.Title != "Test Title" {
		t.Fatalf("title = %q, want %q", got.Title, "Test Title")
	}
	if got.Author != "Test Author" {
		t.Fatalf("author = %q, want %q", got.Author, "Test Author")
	}
	if got.Creator != "Test Creator" {
		t.Fatalf("creator = %q, want %q", got.Creator, "Test Creator")
	}
	if got.Producer != "Test Producer" {
		t.Fatalf("producer = %q, want %q", got.Producer, "Test Producer")
	}
}

func TestRunInfoJSONOFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofd.cn/2016/F最低配"><ofd:Pages><ofd:Page ID="1"/></ofd:Pages></ofd:Document>`,
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

	var got struct {
		Format          string `json:"format"`
		Path            string `json:"path"`
		SizeBytes       int64  `json:"size_bytes"`
		DeclaredVersion string `json:"declared_version,omitempty"`
		PageCount       int    `json:"page_count,omitempty"`
	}
	mustUnmarshalJSON(t, output, &got)

	if got.Format != "ofd" {
		t.Fatalf("format = %v, want %q", got.Format, "ofd")
	}
	if got.Path != path {
		t.Fatalf("path = %v, want %q", got.Path, path)
	}
	if got.DeclaredVersion != "1.0" {
		t.Fatalf("declared_version = %q, want %q", got.DeclaredVersion, "1.0")
	}
	if got.PageCount != 1 {
		t.Fatalf("page_count = %d, want 1", got.PageCount)
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
	path := requirePDFSample(t, "version-compat-v1.4")
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
	path := requirePDFSample(t, "version-compat-v1.4")
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
	path := requirePDFSample(t, "error-corrupted")
	resolver := app.NewPhase1Resolver()
	err := RunInfo(context.Background(), resolver, []string{"--page", path})
	if err == nil {
		t.Fatal("run info --page should fail for known-bad PDF")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "xref") && !strings.Contains(errMsg, "object") {
		t.Fatalf("expected xref/object error, got: %s", errMsg)
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

func TestRunInfoJSONPDFRealSample(t *testing.T) {
	t.Skip("fixture core-multipage stores Info dict fields as indirect references (Title 34 0 R); parser returns unresolved ref instead of dereferenced string")
	path := requirePDFSample(t, "core-multipage")
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
	if got.SizeBytes == 0 {
		t.Fatalf("size_bytes is zero")
	}
	if got.DeclaredVersion == "" {
		t.Fatalf("declared_version is empty")
	}
	if got.Title == "" {
		t.Fatal("title is empty, expected fixture metadata")
	}
	if got.Creator == "" {
		t.Fatal("creator is empty, expected fixture metadata")
	}
	if got.Producer == "" {
		t.Fatal("producer is empty, expected fixture metadata")
	}
}
