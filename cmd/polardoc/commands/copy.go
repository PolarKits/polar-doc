package commands

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/PolarKits/polardoc/internal/app"
	"github.com/PolarKits/polardoc/internal/doc"
)

func parseCopyInput(args []string) (copyInput, error) {
	fs := flag.NewFlagSet("cp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return copyInput{}, err
	}

	if fs.NArg() != 2 {
		return copyInput{}, fmt.Errorf("usage: polardoc cp <src> <dst>")
	}

	return copyInput{
		src: fs.Arg(0),
		dst: fs.Arg(1),
	}, nil
}

type copyInput struct {
	src string
	dst string
}

func RunCopy(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	input, err := parseCopyInput(args)
	if err != nil {
		return err
	}

	format, err := detectFormatByExtension(input.src)
	if err != nil {
		return err
	}

	if format != doc.FormatPDF {
		return fmt.Errorf("save only supported for PDF")
	}

	svc, ok := resolver.ByFormat(format)
	if !ok {
		return fmt.Errorf("no service for format %q", format)
	}

	saver, ok := svc.(app.PDFSaver)
	if !ok {
		return fmt.Errorf("save not supported for format %q", format)
	}

	ref := doc.DocumentRef{
		Format: format,
		Path:   input.src,
	}

	if err := saver.Save(ctx, ref, input.dst); err != nil {
		return err
	}

	fmt.Printf("copied %s to %s\n", input.src, input.dst)
	return nil
}
