package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/PolarKits/polardoc/cmd/polardoc/commands"
	"github.com/PolarKits/polardoc/internal/app"
)

func main() {
	resolver := app.NewPhase1Resolver()
	os.Exit(run(context.Background(), os.Args[1:], resolver, os.Stderr))
}

func run(ctx context.Context, args []string, resolver app.ServiceResolver, stderr io.Writer) int {
	if err := Execute(ctx, args, resolver); err != nil {
		if errors.Is(err, commands.ErrValidationFailed) {
			return 1
		}
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return 0
}
