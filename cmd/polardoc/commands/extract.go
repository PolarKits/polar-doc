package commands

import (
	"context"
	"fmt"

	"github.com/PolarKits/polar-doc/internal/app"
	"github.com/PolarKits/polar-doc/internal/doc"
)

// RunExtract runs the extract command to extract text content from documents.
// It supports PDF and OFD formats, with optional JSON output.
func RunExtract(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	input, err := parseCommandInput("extract", args)
	if err != nil {
		return err
	}

	format, err := detectFormatByExtension(input.path)
	if err != nil {
		if input.json {
			_ = writeJSON(extractErrorResponse{Error: err.Error()})
		}
		return err
	}

	ref := doc.DocumentRef{
		Format: format,
		Path:   input.path,
	}

	svc, ok := resolver.ByFormat(ref.Format)
	if !ok {
		err = fmt.Errorf("no service for format %q", ref.Format)
		if input.json {
			_ = writeJSON(extractErrorResponse{Error: err.Error()})
		}
		return err
	}

	d, err := svc.Open(ctx, ref)
	if err != nil {
		if input.json {
			_ = writeJSON(extractErrorResponse{Error: err.Error()})
		}
		return err
	}
	defer d.Close()

	result, err := svc.ExtractText(ctx, d)
	if err != nil {
		if input.json {
			_ = writeJSON(extractErrorResponse{Error: err.Error()})
		}
		return err
	}

	if input.json {
		return writeJSON(extractResponse{Text: result.Text})
	}

	fmt.Println(result.Text)
	return nil
}

// extractResponse is the JSON response structure for successful text extraction.
type extractResponse struct {
	// Text is the extracted text content.
	Text string `json:"text"`
}

// extractErrorResponse is the JSON response structure for extract command errors.
type extractErrorResponse struct {
	// Error is the error message.
	Error string `json:"error"`
}
