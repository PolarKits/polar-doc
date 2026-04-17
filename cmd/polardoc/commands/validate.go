package commands

import (
	"context"
	"fmt"

	"github.com/PolarKits/polardoc/internal/app"
	"github.com/PolarKits/polardoc/internal/doc"
)

func RunValidate(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	input, err := parseCommandInput("validate", args)
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

	report, err := svc.Validate(ctx, d)
	if err != nil {
		return err
	}

	if input.json {
		return writeJSON(struct {
			Valid  bool     `json:"valid"`
			Errors []string `json:"errors"`
		}{
			Valid:  report.Valid,
			Errors: report.Errors,
		})
	}

	fmt.Printf("valid: %t\n", report.Valid)
	for _, errText := range report.Errors {
		fmt.Printf("error: %s\n", errText)
	}
	return nil
}
