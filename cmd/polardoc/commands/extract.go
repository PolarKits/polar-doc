package commands

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/PolarKits/polar-doc/internal/app"
	"github.com/PolarKits/polar-doc/internal/doc"
)

type extractInput struct {
	path  string
	json  bool
	page  int
}

func parseExtractInput(args []string) (extractInput, error) {
	fs := flag.NewFlagSet("extract", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var file string
	var jsonOutput bool
	var pageNum int
	fs.StringVar(&file, "file", "", "document path")
	fs.StringVar(&file, "f", "", "document path")
	fs.BoolVar(&jsonOutput, "json", false, "print JSON output")
	fs.IntVar(&pageNum, "page", 0, "extract text from a specific page (1-based, 0=all pages)")

	if err := fs.Parse(args); err != nil {
		return extractInput{}, fmt.Errorf("invalid args for extract: %w", err)
	}

	if file == "" {
		if fs.NArg() != 1 {
			return extractInput{}, fmt.Errorf("usage: polardoc extract [--json] [--page N] [--file|-f] <path>")
		}
		file = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return extractInput{}, fmt.Errorf("usage: polardoc extract [--json] [--page N] [--file|-f] <path>")
	}

	return extractInput{
		path: file,
		json: jsonOutput,
		page: pageNum,
	}, nil
}

// RunExtract runs the extract command to extract text content from documents.
// It supports PDF and OFD formats, with optional JSON output and --page flag.
func RunExtract(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	input, err := parseExtractInput(args)
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

	var result doc.TextResult
	if input.page > 0 {
		pagedSvc, ok := svc.(doc.PagedTextExtractor)
		if !ok {
			err = fmt.Errorf("format does not support per-page extraction")
			if input.json {
				_ = writeJSON(extractErrorResponse{Error: err.Error()})
			}
			return err
		}
		result, err = pagedSvc.ExtractTextPage(ctx, d, input.page)
	} else {
		result, err = svc.ExtractText(ctx, d)
	}
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
