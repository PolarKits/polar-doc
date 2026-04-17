package main

import (
	"context"
	"fmt"
	"os"

	"github.com/PolarKits/polardoc/internal/app"
)

func main() {
	resolver := app.NewPhase1Resolver()
	if err := Execute(context.Background(), os.Args[1:], resolver); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
