package main

import (
	"context"
	"fmt"

	"github.com/PolarKits/polardoc/cmd/polardoc/commands"
	"github.com/PolarKits/polardoc/internal/app"
)

func Execute(ctx context.Context, args []string, resolver app.ServiceResolver) error {
	if len(args) == 0 {
		return usageError()
	}

	switch args[0] {
	case "info":
		return commands.RunInfo(ctx, resolver, args[1:])
	case "validate":
		return commands.RunValidate(ctx, resolver, args[1:])
	case "extract":
		return commands.RunExtract(ctx, resolver, args[1:])
	case "help", "-h", "--help":
		return usageError()
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usageText)
	}
}

const usageText = "usage: polardoc <info|validate|extract> [--file|-f] <path>"

func usageError() error {
	return fmt.Errorf(usageText)
}
