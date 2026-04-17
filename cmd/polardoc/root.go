package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/PolarKits/polardoc/cmd/polardoc/commands"
	"github.com/PolarKits/polardoc/internal/app"
)

var errUsage = errors.New("usage")
var errHelp = errors.New("help")

func Execute(ctx context.Context, args []string, resolver app.ServiceResolver) error {
	if len(args) == 0 {
		return errUsage
	}

	switch args[0] {
	case "info":
		return commands.RunInfo(ctx, resolver, args[1:])
	case "validate":
		return commands.RunValidate(ctx, resolver, args[1:])
	case "extract":
		return commands.RunExtract(ctx, resolver, args[1:])
	case "help", "-h", "--help":
		return errHelp
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usageText)
	}
}

const usageText = "polardoc\nusage: polardoc <info|validate> [--file|-f] <path>\ncommands: info, validate"
