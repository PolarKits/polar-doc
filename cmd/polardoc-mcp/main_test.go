package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/PolarKits/polar-doc/internal/app"
	"github.com/PolarKits/polar-doc/internal/mcp"
	fixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

func requirePDFSample(t *testing.T, key string) string {
	t.Helper()
	sample, ok := fixtures.PDFSampleByKey(key)
	if !ok {
		t.Fatalf("missing PDF sample %q", key)
	}
	return sample.Path()
}

// TestMainNormalFlow verifies the full decode→handle→encode pipeline
// using a PDF that supports FirstPageInfo parsing.
func TestMainNormalFlow(t *testing.T) {
	pdfPath := requirePDFSample(t, "standard-pdf20-utf8")

	resolver := app.NewPhase1Resolver()
	firstPageHandler := mcp.NewFirstPageHandler(resolver)
	docInfoHandler := mcp.NewDocumentInfoHandler(resolver)
	validateHandler := mcp.NewDocumentValidateHandler(resolver)

	inputPayload, _ := json.Marshal(mcp.FirstPageInfoInput{Path: pdfPath})
	req := mcpRequest{
		Tool:    mcp.ToolNameFirstPageInfo,
		Payload: inputPayload,
	}

	reqBytes, _ := json.Marshal(req)
	buf := bytes.NewBuffer(reqBytes)
	dec := json.NewDecoder(buf)
	encBuf := &bytes.Buffer{}
	encOut := json.NewEncoder(encBuf)

	if err := dec.Decode(&req); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var result []byte
	var err error
	switch req.Tool {
	case mcp.ToolNameFirstPageInfo:
		result, err = firstPageHandler.Handle(context.Background(), req.Tool, req.Payload)
	case mcp.ToolNameDocumentInfo:
		result, err = docInfoHandler.Handle(context.Background(), req.Tool, req.Payload)
	case mcp.ToolNameDocumentValidate:
		result, err = validateHandler.Handle(context.Background(), req.Tool, req.Payload)
	default:
		err = fmt.Errorf("unknown tool: %s", req.Tool)
	}

	if err != nil {
		t.Fatalf("handle: %v", err)
	}

	if err := encOut.Encode(resultResponse{Result: json.RawMessage(result)}); err != nil {
		t.Fatalf("encode response: %v", err)
	}

	if encBuf.Len() == 0 {
		t.Fatal("expected non-empty response")
	}
}

// TestMainUnknownTool verifies that an unknown tool name returns an error.
func TestMainUnknownTool(t *testing.T) {
	req := mcpRequest{
		Tool:    "unknown_tool",
		Payload: []byte(`{"path":"test.pdf"}`),
	}

	var result []byte
	var err error
	switch req.Tool {
	case mcp.ToolNameFirstPageInfo:
		resolver := app.NewPhase1Resolver()
		handler := mcp.NewFirstPageHandler(resolver)
		result, err = handler.Handle(context.Background(), req.Tool, req.Payload)
	case mcp.ToolNameDocumentInfo:
		resolver := app.NewPhase1Resolver()
		handler := mcp.NewDocumentInfoHandler(resolver)
		result, err = handler.Handle(context.Background(), req.Tool, req.Payload)
	case mcp.ToolNameDocumentValidate:
		resolver := app.NewPhase1Resolver()
		handler := mcp.NewDocumentValidateHandler(resolver)
		result, err = handler.Handle(context.Background(), req.Tool, req.Payload)
	default:
		err = fmt.Errorf("unknown tool: %s", req.Tool)
	}

	if err == nil {
		t.Fatalf("expected error for unknown tool, got result %s", string(result))
	}
	if !bytes.Contains([]byte(err.Error()), []byte("unknown tool")) {
		t.Fatalf("error = %q, want contains 'unknown tool'", err.Error())
	}
}

// TestMainDecodeError verifies that invalid JSON input is detected during decoding.
func TestMainDecodeError(t *testing.T) {
	badJSON := []byte(`{invalid json}`)
	var req mcpRequest
	dec := json.NewDecoder(bytes.NewReader(badJSON))
	if err := dec.Decode(&req); err == nil {
		t.Fatal("expected decode error for invalid JSON")
	}
}

// TestMainHandleError verifies that a handler returns an error when the
// file path is invalid (non-existent PDF).
func TestMainHandleError(t *testing.T) {
	resolver := app.NewPhase1Resolver()
	handler := mcp.NewFirstPageHandler(resolver)

	_, err := handler.Handle(context.Background(), mcp.ToolNameFirstPageInfo, []byte(`{"path":"/nonexistent.pdf"}`))
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}