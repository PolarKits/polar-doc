package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/PolarKits/polar-doc/cmd/polardoc/commands"
	"github.com/PolarKits/polar-doc/internal/app"
)

var errUsage = errors.New("usage")
var errHelp = errors.New("help")

// Execute parses args and dispatches to the appropriate sub-command.
// It returns errUsage or errHelp for usage or help display; all other errors indicate failure.
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
	case "cp":
		return commands.RunCopy(ctx, resolver, args[1:])
	case "help", "-h", "--help":
		return errHelp
	default:
		return fmt.Errorf("unknown command %q\n%s", args[0], usageText)
	}
}

const usageText = "polardoc\nusage: polardoc <info|validate|extract|cp> [--file|-f] <path>\ncommands: info, validate, extract, cp\ninfo flags: --json, --page\nvalidate flags: --json, --deep-validate"
