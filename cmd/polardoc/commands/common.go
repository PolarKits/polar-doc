package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/PolarKits/polardoc/internal/doc"
)

func parseDocumentRef(command string, args []string) (doc.DocumentRef, error) {
	input, err := parseCommandInput(command, args)
	if err != nil {
		return doc.DocumentRef{}, err
	}

	format, err := detectFormatByExtension(input.path)
	if err != nil {
		return doc.DocumentRef{}, err
	}

	return doc.DocumentRef{
		Format: format,
		Path:   input.path,
	}, nil
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
