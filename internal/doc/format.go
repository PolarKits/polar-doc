package doc

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DetectFormatByExtension resolves the format from a file extension.
//
// Routing is currently extension-based and case-insensitive.
// Supported extensions are `.pdf` and `.ofd`.
func DetectFormatByExtension(path string) (Format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".pdf":
		return FormatPDF, nil
	case ".ofd":
		return FormatOFD, nil
	default:
		return "", fmt.Errorf("unsupported file extension for %q", path)
	}
}
