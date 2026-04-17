package commands

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
)

type commandInput struct {
	path string
	json bool
}

func parseCommandInput(command string, args []string) (commandInput, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var file string
	var jsonOutput bool
	fs.StringVar(&file, "file", "", "document path")
	fs.StringVar(&file, "f", "", "document path")
	fs.BoolVar(&jsonOutput, "json", false, "print JSON output")

	if err := fs.Parse(args); err != nil {
		return commandInput{}, fmt.Errorf("invalid args for %s: %w", command, err)
	}

	if file == "" {
		if fs.NArg() != 1 {
			return commandInput{}, fmt.Errorf("usage: polardoc %s [--json] [--file|-f] <path>", command)
		}
		file = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return commandInput{}, fmt.Errorf("usage: polardoc %s [--json] [--file|-f] <path>", command)
	}

	return commandInput{
		path: file,
		json: jsonOutput,
	}, nil
}

func writeJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", data)
	return nil
}
