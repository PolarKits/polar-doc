package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/PolarKits/polar-doc/internal/app"
	"github.com/PolarKits/polar-doc/internal/doc"
)

// ErrValidationFailed indicates document validation failed.
var ErrValidationFailed = errors.New("validation failed")

// RunValidate runs the validate command to check document structural integrity.
// It supports PDF and OFD formats, with optional JSON output.
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
		err := writeJSON(validateResponse{
			Valid:  report.Valid,
			Errors: report.Errors,
		})
		if err != nil {
			return err
		}
		if !report.Valid {
			return ErrValidationFailed
		}
		return nil
	}

	fmt.Printf("valid: %t\n", report.Valid)
	for _, errText := range report.Errors {
		fmt.Printf("error: %s\n", errText)
	}

	if !report.Valid {
		return ErrValidationFailed
	}
	return nil
}

// validateResponse is the JSON response structure for the validate command.
type validateResponse struct {
	// Valid indicates whether the document passed validation.
	Valid bool `json:"valid"`
	// Errors contains validation error messages if Valid is false.
	Errors []string `json:"errors"`
}
