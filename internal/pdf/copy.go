package pdf

import (
	"fmt"
	"io"
	"os"
)

// CopyFile copies a PDF file from src to dst.
//
// This is a minimal write path that performs raw byte copy only.
// It does NOT:
//   - Parse or modify PDF content
//   - Upgrade PDF version
//   - Edit metadata
//   - Perform incremental updates
//
// After CopyFile, the destination file can be opened and processed
// through the normal PDF read pipeline (Open, Validate, FirstPageInfo).
//
// Phase-1 scope: this is the only write capability currently implemented.
// Full PDF editing (content modification, metadata updates, version upgrade)
// requires a separate writer pipeline not yet implemented.
func CopyFile(src, dst string) error {
	if src == "" {
		return fmt.Errorf("CopyFile: source path is empty")
	}
	if dst == "" {
		return fmt.Errorf("CopyFile: destination path is empty")
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("CopyFile: open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("CopyFile: create destination: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("CopyFile: copy: %w", err)
	}

	if err := dstFile.Sync(); err != nil {
		return fmt.Errorf("CopyFile: sync: %w", err)
	}

	return nil
}
