package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/PolarKits/polardoc/internal/app"
	"github.com/PolarKits/polardoc/internal/mcp"
)

func main() {
	resolver := app.NewPhase1Resolver()
	firstPageHandler := mcp.NewFirstPageHandler(resolver)
	docInfoHandler := mcp.NewDocumentInfoHandler(resolver)
	validateHandler := mcp.NewDocumentValidateHandler(resolver)

	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	// MCP protocol loop: read JSON requests from stdin, write responses to stdout.
	// Exit cleanly on EOF when the parent process closes stdin.
	for {
		var req mcpRequest
		if err := dec.Decode(&req); err != nil {
			if err.Error() == "EOF" {
				// Parent process closed stdin; exit gracefully.
				break
			}
			fmt.Fprintf(os.Stderr, "decode error: %v\n", err)
			continue
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
			// Encode error response in MCP protocol format.
			enc.Encode(errorResponse{Error: err.Error()})
			continue
		}

		// Encode successful result in MCP protocol format.
		enc.Encode(resultResponse{Result: json.RawMessage(result)})
	}
}

// errorResponse is the MCP protocol error response structure.
type errorResponse struct {
	// Error is the error message string.
	Error string `json:"error"`
}

// resultResponse is the MCP protocol success response structure.
type resultResponse struct {
	// Result is the JSON-encoded handler result.
	Result json.RawMessage `json:"result"`
}

// mcpRequest is the MCP protocol request structure.
type mcpRequest struct {
	// Tool is the MCP tool name to invoke (e.g. "pdf_first_page_info").
	Tool string `json:"tool"`
	// Payload is the JSON-encoded tool input parameters.
	Payload json.RawMessage `json:"payload"`
}
