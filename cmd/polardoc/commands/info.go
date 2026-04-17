package commands

import (
	"context"
	"fmt"

	"github.com/PolarKits/polardoc/internal/app"
	"github.com/PolarKits/polardoc/internal/doc"
)

func RunInfo(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	input, err := parseCommandInput("info", args)
	if err != nil {
		return err
	}

	format, err := detectFormatByExtension(input.path)
	if err != nil {
		return err
	}

	ref := doc.DocumentRef{
		Format: format,
		Path:   input.path,
	}

	svc, ok := resolver.ByFormat(ref.Format)
	if !ok {
		return fmt.Errorf("no service for format %q", ref.Format)
	}

	d, err := svc.Open(ctx, ref)
	if err != nil {
		return err
	}
	defer d.Close()

	info, err := svc.Info(ctx, d)
	if err != nil {
		return err
	}

	if input.json {
		return writeJSON(struct {
			Format          doc.Format `json:"format"`
			Path            string     `json:"path"`
			SizeBytes       int64      `json:"size_bytes"`
			DeclaredVersion string     `json:"declared_version,omitempty"`
		}{
			Format:          info.Format,
			Path:            info.Path,
			SizeBytes:       info.SizeBytes,
			DeclaredVersion: info.DeclaredVersion,
		})
	}

	fmt.Printf("format: %s\n", info.Format)
	fmt.Printf("path: %s\n", info.Path)
	fmt.Printf("size_bytes: %d\n", info.SizeBytes)
	if info.DeclaredVersion != "" {
		fmt.Printf("declared_version: %s\n", info.DeclaredVersion)
	}
	return nil
}
