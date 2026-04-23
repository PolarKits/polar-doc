package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/PolarKits/polar-doc/internal/app"
)

// ProtocolVersion is the MCP specification version this server implements.
const ProtocolVersion = "2024-11-05"

// jsonrpcRequest represents a JSON-RPC 2.0 request or notification.
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// jsonrpcResponse represents a JSON-RPC 2.0 response.
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

// jsonrpcError represents a JSON-RPC 2.0 error object.
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// InitializeResult is the result of the MCP initialize method.
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

// ServerCapabilities declares the capabilities supported by the server.
type ServerCapabilities struct {
	Tools *ToolCapabilities `json:"tools,omitempty"`
}

// ToolCapabilities declares optional tool-related server capabilities.
type ToolCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ServerInfo identifies the server implementation.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ToolDefinition describes a single tool exposed by the server.
type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolsListResult is the result of the MCP tools/list method.
type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

// toolsCallParams holds the parameters for a tools/call invocation.
type toolsCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolsCallResult is the result of the MCP tools/call method.
type ToolsCallResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock is a single content item within a tool call result.
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Server implements a Model Context Protocol server that communicates over
// stdio using JSON-RPC 2.0. It supports the initialize lifecycle handshake,
// tools/list for service discovery, and tools/call for tool invocation.
type Server struct {
	resolver app.ServiceResolver
	name     string
	version  string
	mu       sync.Mutex
	ready    bool
}

// NewServer creates a new MCP server with the given service resolver,
// server name, and version string.
func NewServer(resolver app.ServiceResolver, name, version string) *Server {
	return &Server{
		resolver: resolver,
		name:     name,
		version:  version,
	}
}

// Serve starts the MCP protocol loop. It reads JSON-RPC requests from r,
// dispatches them to the appropriate handler, and writes responses to w.
// The method blocks until r returns EOF or ctx is cancelled.
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	dec := json.NewDecoder(r)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var req jsonrpcRequest
		if err := dec.Decode(&req); err != nil {
			if err == io.EOF {
				return nil
			}
			continue
		}

		// Notifications carry no ID and expect no response.
		if req.ID == nil {
			s.handleNotification(req.Method)
			continue
		}

		resp := s.dispatch(ctx, req)
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			return fmt.Errorf("encode response: %w", err)
		}
	}
}

// handleNotification processes a JSON-RPC notification that expects no response.
func (s *Server) handleNotification(method string) {
	switch method {
	case "notifications/initialized":
		s.mu.Lock()
		s.ready = true
		s.mu.Unlock()
	}
}

// dispatch routes a JSON-RPC request to the correct handler.
func (s *Server) dispatch(ctx context.Context, req jsonrpcRequest) jsonrpcResponse {
	resp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	var result any
	var rpcErr *jsonrpcError

	switch req.Method {
	case "initialize":
		result = s.handleInitialize()
	case "ping":
		result = struct{}{}
	case "tools/list":
		result = s.handleToolsList()
	case "tools/call":
		result, rpcErr = s.handleToolsCall(ctx, req.Params)
	default:
		rpcErr = &jsonrpcError{
			Code:    -32601,
			Message: fmt.Sprintf("method not found: %s", req.Method),
		}
	}

	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}
	return resp
}

// handleInitialize returns server capabilities and identity information.
func (s *Server) handleInitialize() InitializeResult {
	return InitializeResult{
		ProtocolVersion: ProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolCapabilities{},
		},
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
	}
}

// handleToolsList returns the catalog of tools this server exposes.
func (s *Server) handleToolsList() ToolsListResult {
	pathSchema := map[string]any{
		"type":        "string",
		"description": "File system path to the document.",
	}
	return ToolsListResult{
		Tools: []ToolDefinition{
			{
				Name:        ToolNameFirstPageInfo,
				Description: "Extract first page structure information from a PDF document.",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"path": pathSchema},
					"required":   []string{"path"},
				},
			},
			{
				Name:        ToolNameDocumentInfo,
				Description: "Extract document-level metadata from a PDF or OFD document.",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"path": pathSchema},
					"required":   []string{"path"},
				},
			},
			{
				Name:        ToolNameDocumentValidate,
				Description: "Validate the structural integrity of a PDF or OFD document.",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"path": pathSchema},
					"required":   []string{"path"},
				},
			},
			{
				Name:        ToolNameDocumentExtract,
				Description: "Extract text content from a PDF or OFD document.",
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"path": pathSchema},
					"required":   []string{"path"},
				},
			},
		},
	}
}

// handleToolsCall invokes a tool by name and wraps the result in MCP content format.
func (s *Server) handleToolsCall(ctx context.Context, rawParams json.RawMessage) (any, *jsonrpcError) {
	var params toolsCallParams
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, &jsonrpcError{Code: -32602, Message: fmt.Sprintf("invalid params: %v", err)}
	}

	var handler ToolHandler
	switch params.Name {
	case ToolNameFirstPageInfo:
		handler = NewFirstPageHandler(s.resolver)
	case ToolNameDocumentInfo:
		handler = NewDocumentInfoHandler(s.resolver)
	case ToolNameDocumentValidate:
		handler = NewDocumentValidateHandler(s.resolver)
	case ToolNameDocumentExtract:
		handler = NewDocumentExtractHandler(s.resolver)
	default:
		return nil, &jsonrpcError{Code: -32602, Message: fmt.Sprintf("unknown tool: %s", params.Name)}
	}

	// Re-encode arguments so the existing ToolHandler can unmarshal them.
	payload, _ := json.Marshal(params.Arguments)
	result, err := handler.Handle(ctx, params.Name, payload)
	if err != nil {
		// Tool execution errors are returned as isError results, not JSON-RPC errors.
		return ToolsCallResult{
			Content: []ContentBlock{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return ToolsCallResult{
		Content: []ContentBlock{{Type: "text", Text: string(result)}},
	}, nil
}
