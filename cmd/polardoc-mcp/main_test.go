package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
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

func newTestServer() *mcp.Server {
	return mcp.NewServer(app.NewPhase1Resolver(), "test-polardoc-mcp", "0.0.0-test")
}

// callServer sends a single JSON-RPC request through the server and returns the response.
func callServer(t *testing.T, srv *mcp.Server, method string, params any) map[string]any {
	t.Helper()

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	in := bytes.NewReader(reqBytes)
	out := &bytes.Buffer{}

	if err := srv.Serve(context.Background(), in, out); err != nil {
		t.Fatalf("serve: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("unmarshal response: %v\nraw: %s", err, out.String())
	}

	return resp
}

// TestServerInitialize verifies the MCP initialize handshake returns correct capabilities.
func TestServerInitialize(t *testing.T) {
	srv := newTestServer()
	resp := callServer(t, srv, "initialize", map[string]any{
		"protocolVersion": mcp.ProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "test", "version": "0.1.0"},
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp["result"])
	}

	if result["protocolVersion"] != mcp.ProtocolVersion {
		t.Fatalf("protocolVersion = %v, want %s", result["protocolVersion"], mcp.ProtocolVersion)
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("serverInfo is not a map: %T", result["serverInfo"])
	}
	if serverInfo["name"] != "test-polardoc-mcp" {
		t.Fatalf("serverInfo.name = %v, want test-polardoc-mcp", serverInfo["name"])
	}

	capabilities, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities is not a map: %T", result["capabilities"])
	}
	if _, ok := capabilities["tools"]; !ok {
		t.Fatal("capabilities missing tools")
	}
}

// TestServerPing verifies the ping method returns an empty result.
func TestServerPing(t *testing.T) {
	srv := newTestServer()
	resp := callServer(t, srv, "ping", nil)

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
}

// TestServerToolsList verifies that tools/list returns all three expected tools.
func TestServerToolsList(t *testing.T) {
	srv := newTestServer()
	resp := callServer(t, srv, "tools/list", nil)

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp["result"])
	}

	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools is not a slice: %T", result["tools"])
	}

	if len(tools) != 5 {
		t.Fatalf("len(tools) = %d, want 5", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		tm, ok := tool.(map[string]any)
		if !ok {
			t.Fatalf("tool is not a map: %T", tool)
		}
		name, _ := tm["name"].(string)
		names[name] = true

		if _, ok := tm["inputSchema"]; !ok {
			t.Fatalf("tool %q missing inputSchema", name)
		}
	}

	expected := []string{
		mcp.ToolNameFirstPageInfo,
		mcp.ToolNameDocumentInfo,
		mcp.ToolNameDocumentValidate,
		mcp.ToolNameDocumentExtract,
		mcp.ToolNameDocumentReadPage,
	}
	for _, n := range expected {
		if !names[n] {
			t.Fatalf("missing tool %q in tools/list", n)
		}
	}
}

// TestServerToolsCallFirstPageInfo verifies tools/call for pdf_first_page_info.
func TestServerToolsCallFirstPageInfo(t *testing.T) {
	pdfPath := requirePDFSample(t, "standard-pdf20-utf8")
	srv := newTestServer()

	resp := callServer(t, srv, "tools/call", map[string]any{
		"name":      mcp.ToolNameFirstPageInfo,
		"arguments": map[string]any{"path": pdfPath},
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp["result"])
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content is empty or not a slice: %T", result["content"])
	}

	block, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("content[0] is not a map: %T", content[0])
	}
	if block["type"] != "text" {
		t.Fatalf("content[0].type = %v, want text", block["type"])
	}

	text, _ := block["text"].(string)
	var output map[string]any
	if err := json.Unmarshal([]byte(text), &output); err != nil {
		t.Fatalf("unmarshal tool output: %v\nraw: %s", err, text)
	}
	if output["path"] != pdfPath {
		t.Fatalf("path = %v, want %s", output["path"], pdfPath)
	}
}

// TestServerToolsCallDocumentInfo verifies tools/call for document_info.
func TestServerToolsCallDocumentInfo(t *testing.T) {
	pdfPath := requirePDFSample(t, "standard-pdf20-utf8")
	srv := newTestServer()

	resp := callServer(t, srv, "tools/call", map[string]any{
		"name":      mcp.ToolNameDocumentInfo,
		"arguments": map[string]any{"path": pdfPath},
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp["result"])
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content is empty or not a slice")
	}

	text, _ := content[0].(map[string]any)["text"].(string)
	var output map[string]any
	if err := json.Unmarshal([]byte(text), &output); err != nil {
		t.Fatalf("unmarshal tool output: %v", err)
	}
	if output["format"] != "pdf" {
		t.Fatalf("format = %v, want pdf", output["format"])
	}
}

// TestServerToolsCallDocumentValidate verifies tools/call for document_validate.
func TestServerToolsCallDocumentValidate(t *testing.T) {
	pdfPath := requirePDFSample(t, "standard-pdf20-utf8")
	srv := newTestServer()

	resp := callServer(t, srv, "tools/call", map[string]any{
		"name":      mcp.ToolNameDocumentValidate,
		"arguments": map[string]any{"path": pdfPath},
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp["result"])
	}

	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content is empty or not a slice")
	}

	text, _ := content[0].(map[string]any)["text"].(string)
	var output map[string]any
	if err := json.Unmarshal([]byte(text), &output); err != nil {
		t.Fatalf("unmarshal tool output: %v", err)
	}
	if output["valid"] != true {
		t.Fatalf("valid = %v, want true; errors = %v", output["valid"], output["errors"])
	}
}

// TestServerToolsCallUnknownTool verifies that an unknown tool name returns a JSON-RPC error.
func TestServerToolsCallUnknownTool(t *testing.T) {
	srv := newTestServer()

	resp := callServer(t, srv, "tools/call", map[string]any{
		"name":      "nonexistent_tool",
		"arguments": map[string]any{"path": "/dev/null"},
	})

	if resp["error"] == nil {
		t.Fatal("expected error for unknown tool, got nil")
	}

	rpcErr, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("error is not a map: %T", resp["error"])
	}
	if !strings.Contains(rpcErr["message"].(string), "unknown tool") {
		t.Fatalf("error message = %v, want contains 'unknown tool'", rpcErr["message"])
	}
}

// TestServerMethodNotFound verifies that an unknown method returns a JSON-RPC error.
func TestServerMethodNotFound(t *testing.T) {
	srv := newTestServer()

	resp := callServer(t, srv, "nonexistent/method", nil)

	if resp["error"] == nil {
		t.Fatal("expected error for unknown method, got nil")
	}

	rpcErr, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("error is not a map: %T", resp["error"])
	}
	if !strings.Contains(rpcErr["message"].(string), "method not found") {
		t.Fatalf("error message = %v, want contains 'method not found'", rpcErr["message"])
	}
}

// TestServerNotificationInitialized verifies that the initialized notification
// produces no output (notifications have no response).
func TestServerNotificationInitialized(t *testing.T) {
	srv := newTestServer()

	req := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	in := strings.NewReader(req)
	out := &bytes.Buffer{}

	if err := srv.Serve(context.Background(), in, out); err != nil {
		t.Fatalf("serve: %v", err)
	}

	if out.Len() != 0 {
		t.Fatalf("expected no output for notification, got: %s", out.String())
	}
}

// TestServerToolsCallError verifies that tool execution errors are returned
// as isError results (not JSON-RPC errors).
func TestServerToolsCallError(t *testing.T) {
	srv := newTestServer()

	resp := callServer(t, srv, "tools/call", map[string]any{
		"name":      mcp.ToolNameFirstPageInfo,
		"arguments": map[string]any{"path": "/nonexistent.pdf"},
	})

	if resp["error"] != nil {
		t.Fatalf("unexpected JSON-RPC error: %v", resp["error"])
	}

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("result is not a map: %T", resp["result"])
	}

	if result["isError"] != true {
		t.Fatalf("isError = %v, want true", result["isError"])
	}
}

// TestServerEOF verifies that the server exits cleanly on empty input.
func TestServerEOF(t *testing.T) {
	srv := newTestServer()
	in := strings.NewReader("")
	out := &bytes.Buffer{}

	if err := srv.Serve(context.Background(), in, out); err != nil {
		t.Fatalf("serve on empty input: %v", err)
	}
}
