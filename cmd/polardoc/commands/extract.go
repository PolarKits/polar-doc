package commands

import (
	"context"
	"fmt"

	"github.com/PolarKits/polardoc/internal/app"
	"github.com/PolarKits/polardoc/internal/doc"
)

func RunExtract(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	input, err := parseCommandInput("extract", args)
	if err != nil {
		return err
	}

	if input.json {
		return fmt.Errorf("extract does not support --json flag")
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

	result, err := svc.ExtractText(ctx, d)
	if err != nil {
		return err
	}

	fmt.Println(result.Text)
	return nil
}
