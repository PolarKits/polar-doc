package mcp

import "context"

// ToolHandler is a minimal contract for MCP tool handling.
type ToolHandler interface {
	Handle(ctx context.Context, tool string, payload []byte) ([]byte, error)
}
