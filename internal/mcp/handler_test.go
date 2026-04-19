package mcp

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/PolarKits/polardoc/internal/app"
)

func TestFirstPageHandlerPDFSuccess(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.5.x.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testPDF_Version.5.x.pdf not found")
	}

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
	path := filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.8.x.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("testPDF_Version.8.x.pdf not found")
	}

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
		path        string
		wantSuccess bool
	}{
		{"pdf20-utf8", filepath.Join("..", "..", "testdata", "pdf", "pdf20-utf8-test.pdf"), true},
		{"redhat-openshift", filepath.Join("..", "..", "testdata", "pdf", "Red_Hat_OpenShift_Serverless-1.35-Serverless_Logic-en-US.pdf"), true},
		{"sample-local-pdf", filepath.Join("..", "..", "testdata", "pdf", "sample-local-pdf.pdf"), true},
		{"testPDF-5x", filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.5.x.pdf"), true},
		{"testPDF-8x", filepath.Join("..", "..", "testdata", "pdf", "testPDF_Version.8.x.pdf"), false},
	}

	for _, tc := range samples {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := os.Stat(tc.path); os.IsNotExist(err) {
				t.Skipf("%s not found", tc.name)
			}

			input := FirstPageInfoInput{Path: tc.path}
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
	path := filepath.Join("..", "..", "testdata", "pdf", "pdf20-utf8-test.pdf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Skip("pdf20-utf8-test.pdf not found")
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
