package commands

import (
	"context"
	"fmt"

	"github.com/PolarKits/polardoc/internal/app"
)

func RunValidate(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	ref, err := parseDocumentRef("validate", args)
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

	report, err := svc.Validate(ctx, d)
	if err != nil {
		return err
	}

	fmt.Printf("valid: %t\n", report.Valid)
	for _, errText := range report.Errors {
		fmt.Printf("error: %s\n", errText)
	}
	return nil
}
