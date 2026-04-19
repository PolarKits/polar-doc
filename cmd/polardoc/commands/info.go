package commands

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/PolarKits/polardoc/internal/app"
	"github.com/PolarKits/polardoc/internal/doc"
)

type infoInput struct {
	path string
	json bool
	page bool
}

type pageRef struct {
	ObjNum int64 `json:"obj_num"`
	GenNum int64 `json:"gen_num"`
}

type pageInfoOutput struct {
	Path      string    `json:"path"`
	PagesRef  pageRef   `json:"pages_ref"`
	PageRef   pageRef   `json:"page_ref"`
	Parent    pageRef   `json:"parent"`
	MediaBox  []float64 `json:"media_box"`
	Resources pageRef   `json:"resources"`
	Contents  []pageRef `json:"contents"`
	Rotate    *int64    `json:"rotate,omitempty"`
}

func parseInfoInput(args []string) (infoInput, error) {
	fs := flag.NewFlagSet("info", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var file string
	var jsonOutput, pageInfo bool
	fs.StringVar(&file, "file", "", "document path")
	fs.StringVar(&file, "f", "", "document path")
	fs.BoolVar(&jsonOutput, "json", false, "print JSON output")
	fs.BoolVar(&pageInfo, "page", false, "show first page info")

	if err := fs.Parse(args); err != nil {
		return infoInput{}, fmt.Errorf("invalid args for info: %w", err)
	}

	if file == "" {
		if fs.NArg() != 1 {
			return infoInput{}, fmt.Errorf("usage: polardoc info [--json] [--page] [--file|-f] <path>")
		}
		file = fs.Arg(0)
	} else if fs.NArg() != 0 {
		return infoInput{}, fmt.Errorf("usage: polardoc info [--json] [--page] [--file|-f] <path>")
	}

	return infoInput{
		path: file,
		json: jsonOutput,
		page: pageInfo,
	}, nil
}

func RunInfo(ctx context.Context, resolver app.ServiceResolver, args []string) error {
	input, err := parseInfoInput(args)
	if err != nil {
		return err
	}

	format, err := detectFormatByExtension(input.path)
	if err != nil {
		return err
	}

	if input.page {
		if format != doc.FormatPDF {
			return fmt.Errorf("--page is only supported for PDF")
		}
		return runInfoPage(input, resolver)
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

	info, err := svc.Info(ctx, d)
	if err != nil {
		return err
	}

	if input.json {
		return writeJSON(struct {
			Format          doc.Format `json:"format"`
			Path            string     `json:"path"`
			SizeBytes       int64      `json:"size_bytes"`
			DeclaredVersion string     `json:"declared_version,omitempty"`
			PageCount       int        `json:"page_count,omitempty"`
			FileIdentifiers []string    `json:"file_identifiers,omitempty"`
			Title           string     `json:"title,omitempty"`
			Author          string     `json:"author,omitempty"`
			Creator         string     `json:"creator,omitempty"`
			Producer        string     `json:"producer,omitempty"`
		}{
			Format:          info.Format,
			Path:            info.Path,
			SizeBytes:       info.SizeBytes,
			DeclaredVersion: info.DeclaredVersion,
			PageCount:       info.PageCount,
			FileIdentifiers: info.FileIdentifiers,
			Title:           info.Title,
			Author:          info.Author,
			Creator:         info.Creator,
			Producer:        info.Producer,
		})
	}

	fmt.Printf("format: %s\n", info.Format)
	fmt.Printf("path: %s\n", info.Path)
	fmt.Printf("size_bytes: %d\n", info.SizeBytes)
	if info.DeclaredVersion != "" {
		fmt.Printf("declared_version: %s\n", info.DeclaredVersion)
	}
	return nil
}

func runInfoPage(input infoInput, resolver app.ServiceResolver) error {
	svc, ok := resolver.ByFormat(doc.FormatPDF)
	if !ok {
		return fmt.Errorf("no service for format %q", doc.FormatPDF)
	}

	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: input.path})
	if err != nil {
		return err
	}
	defer d.Close()

	result, err := svc.FirstPageInfo(context.Background(), d)
	if err != nil {
		return err
	}

	if input.json {
		out := pageInfoOutput{
			Path: input.path,
			PagesRef: pageRef{
				ObjNum: result.PagesRef.ObjNum,
				GenNum: result.PagesRef.GenNum,
			},
			PageRef: pageRef{
				ObjNum: result.PageRef.ObjNum,
				GenNum: result.PageRef.GenNum,
			},
			Parent: pageRef{
				ObjNum: result.Parent.ObjNum,
				GenNum: result.Parent.GenNum,
			},
			MediaBox: result.MediaBox,
			Resources: pageRef{
				ObjNum: result.Resources.ObjNum,
				GenNum: result.Resources.GenNum,
			},
			Contents: refsToPageRefs(result.Contents),
			Rotate:   result.Rotate,
		}
		return writeJSON(out)
	}

	fmt.Printf("path: %s\n", input.path)
	fmt.Printf("pages_ref: %d %d R\n", result.PagesRef.ObjNum, result.PagesRef.GenNum)
	fmt.Printf("page_ref: %d %d R\n", result.PageRef.ObjNum, result.PageRef.GenNum)
	fmt.Printf("parent: %d %d R\n", result.Parent.ObjNum, result.Parent.GenNum)
	fmt.Printf("media_box: %v\n", result.MediaBox)
	fmt.Printf("resources: %d %d R\n", result.Resources.ObjNum, result.Resources.GenNum)
	if len(result.Contents) > 0 {
		fmt.Printf("contents: ")
		for i, c := range result.Contents {
			if i > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%d %d R", c.ObjNum, c.GenNum)
		}
		fmt.Printf("\n")
	}
	if result.Rotate != nil {
		fmt.Printf("rotate: %d\n", *result.Rotate)
	}
	return nil
}

func refsToPageRefs(refs []doc.RefInfo) []pageRef {
	result := make([]pageRef, len(refs))
	for i, r := range refs {
		result[i] = pageRef{r.ObjNum, r.GenNum}
	}
	return result
}
