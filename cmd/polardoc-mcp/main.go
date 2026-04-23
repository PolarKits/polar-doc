package main

import (
	"context"
	"fmt"
	"os"

	"github.com/PolarKits/polar-doc/internal/app"
	"github.com/PolarKits/polar-doc/internal/mcp"
)

func main() {
	resolver := app.NewPhase1Resolver()
	server := mcp.NewServer(resolver, "polardoc-mcp", "0.1.0")
	if err := server.Serve(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
