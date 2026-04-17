package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/PolarKits/polardoc/internal/app"
)

func main() {
	resolver := app.NewPhase1Resolver()
	os.Exit(run(context.Background(), os.Args[1:], resolver, os.Stderr))
}

func run(ctx context.Context, args []string, resolver app.ServiceResolver, stderr io.Writer) int {
	if err := Execute(ctx, args, resolver); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}
