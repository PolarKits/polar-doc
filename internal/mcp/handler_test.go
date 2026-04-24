package mcp

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/app"
	"github.com/PolarKits/polar-doc/internal/doc"
	testfixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

func TestFirstPageHandlerPDFSuccess(t *testing.T) {
	path := requirePDFSample(t, "standard-pdf20-utf8")

	resolver := app.NewPhase1Resolver()
	handler := NewFirstPageHandler(resolver)

	input := FirstPageInfoInput{Path: path}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameFirstPageInfo, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var output FirstPageInfoOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if output.Path != path {
		t.Fatalf("path = %q, want %q", output.Path, path)
	}
	if output.PagesRef.ObjNum == 0 {
		t.Fatalf("pages_ref obj_num is zero")
	}
	if output.PageRef.ObjNum == 0 {
		t.Fatalf("page_ref obj_num is zero")
	}
	if output.Parent.ObjNum == 0 {
		t.Fatalf("parent obj_num is zero")
	}
	if len(output.MediaBox) == 0 {
		t.Fatalf("media_box is empty")
	}
}

func TestFirstPageHandlerKnownBadPDF(t *testing.T) {
	path := requirePDFSample(t, "version-compat-v1.7")

	resolver := app.NewPhase1Resolver()
	handler := NewFirstPageHandler(resolver)

	input := FirstPageInfoInput{Path: path}
	payload, _ := json.Marshal(input)

	_, err := handler.Handle(context.Background(), ToolNameFirstPageInfo, payload)
	if err == nil {
		t.Fatal("handler should fail for known-bad PDF")
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Fatalf("error message is empty")
	}
	t.Logf("known-bad PDF correctly fails: %s", errMsg)
}

func TestFirstPageHandlerOFDUnsupported(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewFirstPageHandler(resolver)

	input := FirstPageInfoInput{Path: "/fake/sample.ofd"}
	payload, _ := json.Marshal(input)

	_, err := handler.Handle(context.Background(), ToolNameFirstPageInfo, payload)
	if err == nil {
		t.Fatal("handler should fail for OFD")
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Fatalf("error message is empty")
	}
}

func TestFirstPageHandlerUnknownTool(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewFirstPageHandler(resolver)

	_, err := handler.Handle(context.Background(), "unknown_tool", []byte("{}"))
	if err == nil {
		t.Fatal("handler should fail for unknown tool")
	}
}

func TestFirstPageHandlerEmptyPath(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewFirstPageHandler(resolver)

	input := FirstPageInfoInput{Path: ""}
	payload, _ := json.Marshal(input)

	_, err := handler.Handle(context.Background(), ToolNameFirstPageInfo, payload)
	if err == nil {
		t.Fatal("handler should fail for empty path")
	}
}

func TestFirstPageHandlerInvalidJSON(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewFirstPageHandler(resolver)

	_, err := handler.Handle(context.Background(), ToolNameFirstPageInfo, []byte("invalid json"))
	if err == nil {
		t.Fatal("handler should fail for invalid JSON")
	}
}

func TestFirstPageHandlerPDFMatrix(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewFirstPageHandler(resolver)

	samples := []struct {
		name        string
		key         string
		wantSuccess bool
	}{
		{"standard-pdf20-utf8", "standard-pdf20-utf8", true},
		{"version-compat-v1.4", "version-compat-v1.4", true},
		{"standard-pdfa-archival", "standard-pdfa-archival", true},
		{"version-compat-v1.7", "version-compat-v1.7", false},
		{"error-corrupted", "error-corrupted", false},
	}

	for _, tc := range samples {
		t.Run(tc.name, func(t *testing.T) {
			path := requirePDFSample(t, tc.key)

			input := FirstPageInfoInput{Path: path}
			payload, _ := json.Marshal(input)

			result, err := handler.Handle(context.Background(), ToolNameFirstPageInfo, payload)
			if tc.wantSuccess {
				if err != nil {
					t.Fatalf("handler error: %v", err)
				}
				var output FirstPageInfoOutput
				if err := json.Unmarshal(result, &output); err != nil {
					t.Fatalf("unmarshal result: %v", err)
				}
				if output.PagesRef.ObjNum == 0 {
					t.Fatalf("pages_ref obj_num is zero")
				}
			} else {
				if err == nil {
					t.Fatal("expected error for known-bad PDF")
				}
				t.Logf("correctly failed: %v", err)
			}
		})
	}
}

func TestDocumentInfoHandlerPDFWithMetadata(t *testing.T) {
	path := requirePDFSample(t, "standard-pdf20-utf8")

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentInfoHandler(resolver)

	input := DocumentInfoInput{Path: path}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameDocumentInfo, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var output DocumentInfoOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if output.Format != "pdf" {
		t.Fatalf("format = %q, want %q", output.Format, "pdf")
	}
	if output.Path != path {
		t.Fatalf("path = %q, want %q", output.Path, path)
	}
	if output.SizeBytes == 0 {
		t.Fatalf("size_bytes is zero")
	}
	if output.DeclaredVersion == "" {
		t.Fatalf("declared_version is empty")
	}
	t.Logf("declared_version = %q", output.DeclaredVersion)
	if output.Title != "" {
		t.Logf("title = %q", output.Title)
	}
	if output.Author != "" {
		t.Logf("author = %q", output.Author)
	}
	if output.Creator != "" {
		t.Logf("creator = %q", output.Creator)
	}
	if output.Producer != "" {
		t.Logf("producer = %q", output.Producer)
	}
	if len(output.FileIdentifiers) > 0 {
		t.Logf("file_identifiers = %v", output.FileIdentifiers)
	}
}

func TestDocumentInfoHandlerOFD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.ofd")
	content := buildOFDPackage(t, map[string]string{
		"OFD.xml":            `<?xml version="1.0" encoding="UTF-8"?><ofd><Version>1.0</Version><DocRoot>Doc_0/Document.xml</DocRoot></ofd>`,
		"Doc_0/Document.xml": `<?xml version="1.0" encoding="UTF-8"?><ofd:Document xmlns:ofd="http://www.ofd.cn/2016/F最低配"><ofd:Pages><ofd:Page ID="1"/><ofd:Page ID="2"/></ofd:Pages></ofd:Document>`,
	})
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write sample OFD: %v", err)
	}

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentInfoHandler(resolver)

	input := DocumentInfoInput{Path: path}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameDocumentInfo, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var output DocumentInfoOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if output.Format != "ofd" {
		t.Fatalf("format = %q, want %q", output.Format, "ofd")
	}
	if output.DeclaredVersion != "1.0" {
		t.Fatalf("declared_version = %q, want %q", output.DeclaredVersion, "1.0")
	}
	if output.PageCount != 2 {
		t.Fatalf("page_count = %d, want 2", output.PageCount)
	}
}

func TestDocumentInfoHandlerUnknownTool(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewDocumentInfoHandler(resolver)

	_, err := handler.Handle(context.Background(), "unknown_tool", []byte("{}"))
	if err == nil {
		t.Fatal("handler should fail for unknown tool")
	}
}

func TestDocumentInfoHandlerEmptyPath(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewDocumentInfoHandler(resolver)

	input := DocumentInfoInput{Path: ""}
	payload, _ := json.Marshal(input)

	_, err := handler.Handle(context.Background(), ToolNameDocumentInfo, payload)
	if err == nil {
		t.Fatal("handler should fail for empty path")
	}
}

func TestDocumentValidateHandlerPDFSuccess(t *testing.T) {
	path := requirePDFSample(t, "standard-pdf20-utf8")

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentValidateHandler(resolver)

	input := DocumentValidateInput{Path: path}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameDocumentValidate, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var output DocumentValidateOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if !output.Valid {
		t.Fatalf("valid = false, want true; errors = %v", output.Errors)
	}
}

func TestDocumentValidateHandlerJSONFieldNames(t *testing.T) {
	path := requirePDFSample(t, "standard-pdf20-utf8")

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentValidateHandler(resolver)

	input := DocumentValidateInput{Path: path}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameDocumentValidate, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	// Verify the raw JSON uses lowercase keys per the API contract.
	// doc.ValidationReport has no JSON tags, so a direct marshal would
	// emit {"Valid":...,"Errors":...}; we must ensure that does not happen.
	if bytes.Contains(result, []byte("\"Valid\"")) {
		t.Fatalf("JSON contains uppercase \"Valid\", want lowercase: %s", string(result))
	}
	if !bytes.Contains(result, []byte("\"valid\"")) {
		t.Fatalf("JSON missing lowercase \"valid\": %s", string(result))
	}
}

func TestDocumentValidateHandlerOFDSuccess(t *testing.T) {
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
	handler := NewDocumentValidateHandler(resolver)

	input := DocumentValidateInput{Path: path}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameDocumentValidate, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var output DocumentValidateOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if !output.Valid {
		t.Fatalf("valid = false, want true; errors = %v", output.Errors)
	}
}

func TestDocumentValidateHandlerEmptyPath(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewDocumentValidateHandler(resolver)

	input := DocumentValidateInput{Path: ""}
	payload, _ := json.Marshal(input)

	_, err := handler.Handle(context.Background(), ToolNameDocumentValidate, payload)
	if err == nil {
		t.Fatal("handler should fail for empty path")
	}
}

func TestDocumentValidateHandlerUnsupportedExtension(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewDocumentValidateHandler(resolver)

	input := DocumentValidateInput{Path: "/path/to/file.txt"}
	payload, _ := json.Marshal(input)

	_, err := handler.Handle(context.Background(), ToolNameDocumentValidate, payload)
	if err == nil {
		t.Fatal("handler should fail for unsupported extension")
	}
}

func TestDocumentValidateHandlerUnknownTool(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewDocumentValidateHandler(resolver)

	_, err := handler.Handle(context.Background(), "unknown_tool", []byte("{}"))
	if err == nil {
		t.Fatal("handler should fail for unknown tool")
	}
}

// TestValidateInputPathRejectsTraversal verifies that a path containing ".."
// traversal components is rejected by the MCP handlers before any file access.
func TestValidateInputPathRejectsTraversal(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewDocumentValidateHandler(resolver)

	input := DocumentValidateInput{Path: "../../etc/passwd.pdf"}
	payload, _ := json.Marshal(input)

	_, err := handler.Handle(context.Background(), ToolNameDocumentValidate, payload)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "traversal")
	}
}

// TestValidateInputPathAcceptsNormalPaths verifies that benign paths without
// traversal components are accepted by validateInputPath.
func TestValidateInputPathAcceptsNormalPaths(t *testing.T) {
	tests := []struct {
		path string
	}{
		{"/home/user/doc.pdf"},
		{"./doc.pdf"},
		{"/foo/bar/../baz.pdf"}, // Clean resolves this to /foo/baz, so it is safe
		{"testdata/sample.ofd"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if err := validateInputPath(tc.path); err != nil {
				t.Fatalf("validateInputPath(%q): unexpected error: %v", tc.path, err)
			}
		})
	}
}

func TestDetectFormatByExtensionUppercase(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
	}{
		{"/path/to/file.PDF", false},
		{"/path/to/file.OFD", false},
		{"/path/to/file.Pdf", false},
		{"/path/to/file.Ofd", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			format, err := doc.DetectFormatByExtension(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tc.path)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tc.path, err)
				}
				if format != doc.FormatPDF && format != doc.FormatOFD {
					t.Errorf("unexpected format %q for %q", format, tc.path)
				}
			}
		})
	}
}

func TestDetectFormatByExtensionEdgeCases(t *testing.T) {
	tests := []struct {
		path    string
		wantErr bool
		errMsg  string
	}{
		{"", true, "unsupported"},
		{"/a", true, "unsupported"},
		{"/ab", true, "unsupported"},
		{"/abc", true, "unsupported"},
		{"/path", true, "unsupported"},
		{"/path/noext", true, "unsupported"},
		{"/path/file.txt", true, "unsupported"},
		{"/path/file.xyz", true, "unsupported"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			_, err := doc.DetectFormatByExtension(tc.path)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error for %q", tc.path)
				} else if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("error %q for %q does not contain %q", err.Error(), tc.path, tc.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tc.path, err)
				}
			}
		})
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

func TestDocumentExtractHandlerPDF(t *testing.T) {
	path := requirePDFSample(t, "standard-pdf20-utf8")

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentExtractHandler(resolver)

	input := DocumentExtractInput{Path: path}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameDocumentExtract, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var output DocumentExtractOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if output.Path != path {
		t.Fatalf("path = %q, want %q", output.Path, path)
	}
	if output.Text == "" {
		t.Fatal("text is empty, expected non-empty")
	}
	if output.PageCount <= 0 {
		t.Fatalf("page_count = %d, want > 0", output.PageCount)
	}
	t.Logf("text length = %d, page_count = %d", len(output.Text), output.PageCount)
}

func TestDocumentExtractHandlerOFD(t *testing.T) {
	sample, ok := testfixtures.OFDSampleByKey("core-multipage")
	if !ok {
		t.Fatal("core-multipage OFD sample not found")
	}
	path := sample.Path()

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentExtractHandler(resolver)

	input := DocumentExtractInput{Path: path}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameDocumentExtract, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var output DocumentExtractOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if output.Path != path {
		t.Fatalf("path = %q, want %q", output.Path, path)
	}
	if output.Text == "" {
		t.Fatal("text is empty, expected non-empty")
	}
	if output.PageCount <= 0 {
		t.Fatalf("page_count = %d, want > 0", output.PageCount)
	}
	t.Logf("text length = %d, page_count = %d", len(output.Text), output.PageCount)
}

func TestDocumentExtractHandlerEmptyPath(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewDocumentExtractHandler(resolver)

	input := DocumentExtractInput{Path: ""}
	payload, _ := json.Marshal(input)

	_, err := handler.Handle(context.Background(), ToolNameDocumentExtract, payload)
	if err == nil {
		t.Fatal("handler should fail for empty path")
	}
}

func TestDocumentExtractHandlerUnsupportedExtension(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := NewDocumentExtractHandler(resolver)

	input := DocumentExtractInput{Path: "/path/to/file.txt"}
	payload, _ := json.Marshal(input)

	_, err := handler.Handle(context.Background(), ToolNameDocumentExtract, payload)
	if err == nil {
		t.Fatal("handler should fail for unsupported extension")
	}
}

func TestDocumentReadPageHandler_PDF(t *testing.T) {
	path := requirePDFSample(t, "core-multipage")

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentReadPageHandler(resolver)

	input := DocumentReadPageInput{Path: path, Page: 1}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameDocumentReadPage, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var output DocumentReadPageOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if output.Path != path {
		t.Fatalf("path = %q, want %q", output.Path, path)
	}
	if output.Page != 1 {
		t.Fatalf("page = %d, want 1", output.Page)
	}
	if output.TotalPages <= 0 {
		t.Fatalf("total_pages = %d, want > 0", output.TotalPages)
	}
	if output.ObjRef == "" {
		t.Fatal("obj_ref is empty, expected non-empty")
	}
	if output.ContentSize <= 0 {
		t.Fatalf("content_size = %d, want > 0", output.ContentSize)
	}
}

func TestDocumentReadPageHandler_OFD(t *testing.T) {
	path := requireOFDSample(t, "core-multipage")

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentReadPageHandler(resolver)

	input := DocumentReadPageInput{Path: path, Page: 1}
	payload, _ := json.Marshal(input)

	result, err := handler.Handle(context.Background(), ToolNameDocumentReadPage, payload)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}

	var output DocumentReadPageOutput
	if err := json.Unmarshal(result, &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if output.Path != path {
		t.Fatalf("path = %q, want %q", output.Path, path)
	}
	if output.Page != 1 {
		t.Fatalf("page = %d, want 1", output.Page)
	}
	if output.TotalPages <= 0 {
		t.Fatalf("total_pages = %d, want > 0", output.TotalPages)
	}
	if output.ObjRef == "" {
		t.Fatal("obj_ref is empty, expected non-empty")
	}
	if output.ContentSize <= 0 {
		t.Fatalf("content_size = %d, want > 0", output.ContentSize)
	}
}

func TestDocumentReadPageHandler_OutOfRange(t *testing.T) {
	path := requirePDFSample(t, "core-multipage")

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentReadPageHandler(resolver)

	_, err := handler.Handle(context.Background(), ToolNameDocumentReadPage, []byte(`{"path":`+`"`+path+`","page":9999}`))
	if err == nil {
		t.Fatal("expected error for out-of-range page, got nil")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("error = %q, want contains 'out of range'", err.Error())
	}
}

func TestDocumentReadPageHandler_InvalidPage(t *testing.T) {
	path := requirePDFSample(t, "core-multipage")

	resolver := app.NewPhase1Resolver()
	handler := NewDocumentReadPageHandler(resolver)

	_, err := handler.Handle(context.Background(), ToolNameDocumentReadPage, []byte(`{"path":`+`"`+path+`","page":0}`))
	if err == nil {
		t.Fatal("handler should fail for page=0")
	}
	if !strings.Contains(err.Error(), "page must be >= 1") {
		t.Fatalf("error = %q, want contains 'page must be >= 1'", err.Error())
	}
}
