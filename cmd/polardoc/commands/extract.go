package commands

import (
	"context"
	"fmt"

	"github.com/PolarKits/polardoc/internal/app"
)

func RunExtract(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	ref, err := parseDocumentRef("extract", args)
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

	result, err := svc.ExtractText(ctx, d)
	if err != nil {
		return err
	}

	fmt.Println(result.Text)
	return nil
}
