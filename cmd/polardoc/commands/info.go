package commands

import (
	"context"
	"fmt"

	"github.com/PolarKits/polardoc/internal/app"
)

func RunInfo(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	ref, err := parseDocumentRef("info", args)
	if err != nil {
		return err
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

	fmt.Printf("format: %s\n", info.Format)
	fmt.Printf("path: %s\n", info.Path)
	fmt.Printf("size_bytes: %d\n", info.SizeBytes)
	if info.DeclaredVersion != "" {
		fmt.Printf("declared_version: %s\n", info.DeclaredVersion)
	}
	return nil
}
