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

	dec := json.NewDecoder(os.Stdin)
	enc := json.NewEncoder(os.Stdout)

	for {
		var req struct {
			Tool    string          `json:"tool"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := dec.Decode(&req); err != nil {
			if err.Error() == "EOF" {
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
		default:
			err = fmt.Errorf("unknown tool: %s", req.Tool)
		}

		if err != nil {
			enc.Encode(struct {
				Error string `json:"error"`
			}{Error: err.Error()})
			continue
		}

		enc.Encode(struct {
			Result json.RawMessage `json:"result"`
		}{Result: json.RawMessage(result)})
	}
}
