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

	format, err := detectFormatByExtension(input.path)
	if err != nil {
		if input.json {
			_ = writeJSON(struct {
				Error string `json:"error"`
			}{Error: err.Error()})
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
			_ = writeJSON(struct {
				Error string `json:"error"`
			}{Error: err.Error()})
		}
		return err
	}

	d, err := svc.Open(ctx, ref)
	if err != nil {
		if input.json {
			_ = writeJSON(struct {
				Error string `json:"error"`
			}{Error: err.Error()})
		}
		return err
	}
	defer d.Close()

	result, err := svc.ExtractText(ctx, d)
	if err != nil {
		if input.json {
			_ = writeJSON(struct {
				Error string `json:"error"`
			}{Error: err.Error()})
		}
		return err
	}

	if input.json {
		return writeJSON(struct {
			Text string `json:"text"`
		}{Text: result.Text})
	}

	fmt.Println(result.Text)
	return nil
}
