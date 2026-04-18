package mcp

import (
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
