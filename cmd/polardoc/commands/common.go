package commands

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/PolarKits/polardoc/internal/doc"
)

func parseDocumentRef(command string, args []string) (doc.DocumentRef, error) {
	path, err := parsePathArg(command, args)
	if err != nil {
		return doc.DocumentRef{}, err
	}

	format, err := detectFormatByExtension(path)
	if err != nil {
		return doc.DocumentRef{}, err
	}

	return doc.DocumentRef{
		Format: format,
		Path:   path,
	}, nil
}

func parsePathArg(command string, args []string) (string, error) {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var file string
	fs.StringVar(&file, "file", "", "document path")
	fs.StringVar(&file, "f", "", "document path")

	if err := fs.Parse(args); err != nil {
		return "", fmt.Errorf("invalid args for %s: %w", command, err)
	}

	if file == "" {
		if fs.NArg() != 1 {
			return "", fmt.Errorf("usage: polardoc %s [--file|-f] <path>", command)
		}
		file = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return "", fmt.Errorf("usage: polardoc %s [--file|-f] <path>", command)
	}

	return file, nil
}

func detectFormatByExtension(path string) (doc.Format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return doc.FormatPDF, nil
	case ".ofd":
		return doc.FormatOFD, nil
	default:
		return "", fmt.Errorf("unsupported file extension for %q", path)
	}
}
