package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/PolarKits/polar-doc/internal/app"
	"github.com/PolarKits/polar-doc/internal/doc"
)

type infoInput struct {
	path string
	json bool
	page bool
}

type pageRef struct {
	// ObjNum is the object number of the indirect reference.
	ObjNum int64 `json:"obj_num"`
	// GenNum is the generation number of the indirect reference.
	GenNum int64 `json:"gen_num"`
}

type pageInfoOutput struct {
	// Path is the file system path to the document.
	Path string `json:"path"`
	// PagesRef is the indirect reference to the root Pages object.
	PagesRef pageRef `json:"pages_ref"`
	// PageRef is the indirect reference to the first page object.
	PageRef pageRef `json:"page_ref"`
	// Parent is the indirect reference to the parent Pages object.
	Parent pageRef `json:"parent"`
	// MediaBox is the page media box rectangle [llx, lly, urx, ury].
	MediaBox []float64 `json:"media_box"`
	// Resources is the indirect reference to the resource dictionary.
	Resources pageRef `json:"resources"`
	// Contents is a slice of indirect references to content streams.
	Contents []pageRef `json:"contents"`
	// Rotate is the page rotation in degrees (0, 90, 180, 270). Nil if not specified.
	Rotate *int64 `json:"rotate,omitempty"`
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

// RunInfo runs the info command to retrieve document metadata.
// It supports PDF and OFD formats, with optional JSON output and first page info for PDF.
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
		var creationDate, modDate *time.Time
		if !info.CreationDate.IsZero() {
			t := info.CreationDate
			creationDate = &t
		}
		if !info.ModDate.IsZero() {
			t := info.ModDate
			modDate = &t
		}
		return writeJSON(infoResponse{
			Format:             info.Format,
			Path:               info.Path,
			SizeBytes:          info.SizeBytes,
			DeclaredVersion:    info.DeclaredVersion,
			PageCount:          info.PageCount,
			FileIdentifiers:    info.FileIdentifiers,
			Title:              info.Title,
			Author:             info.Author,
			Creator:            info.Creator,
			Producer:           info.Producer,
			CreationDate:       creationDate,
			ModDate:            modDate,
			IsEncrypted:        info.IsEncrypted,
			EncryptionAlgorithm: info.EncryptionAlgorithm,
			Seals:             info.Seals,
			Fonts:             info.Fonts,
			MediaFiles:         info.MediaFiles,
			Pages:              info.Pages,
			Annotations:        info.Annotations,
		})
	}

	fmt.Printf("format: %s\n", info.Format)
	fmt.Printf("path: %s\n", info.Path)
	fmt.Printf("size_bytes: %d\n", info.SizeBytes)
	if info.DeclaredVersion != "" {
		fmt.Printf("declared_version: %s\n", info.DeclaredVersion)
	}
	if !info.CreationDate.IsZero() {
		fmt.Printf("creation_date: %s\n", info.CreationDate.Format(time.RFC3339))
	}
	if !info.ModDate.IsZero() {
		fmt.Printf("mod_date: %s\n", info.ModDate.Format(time.RFC3339))
	}
	if info.IsEncrypted {
		fmt.Printf("encrypted: true (%s)\n", info.EncryptionAlgorithm)
	}
	return nil
}

// infoResponse is the JSON response structure for the info command.
// It mirrors doc.InfoResult but uses JSON tags suitable for direct serialization.
type infoResponse struct {
	// Format is the document format domain (PDF or OFD).
	Format doc.Format `json:"format"`
	// Path is the file system path to the document.
	Path string `json:"path"`
	// SizeBytes is the file size in bytes.
	SizeBytes int64 `json:"size_bytes"`
	// DeclaredVersion is the format version declared in the document header.
	DeclaredVersion string `json:"declared_version,omitempty"`
	// PageCount is the number of pages in the document.
	PageCount int `json:"page_count,omitempty"`
	// FileIdentifiers is the list of file identifiers (PDF only).
	FileIdentifiers []string `json:"file_identifiers,omitempty"`
	// Title is the document title from metadata (PDF only).
	Title string `json:"title,omitempty"`
	// Author is the document author from metadata (PDF only).
	Author string `json:"author,omitempty"`
	// Creator is the document creator from metadata (PDF only).
	Creator string `json:"creator,omitempty"`
	// Producer is the document producer from metadata (PDF only).
	Producer string `json:"producer,omitempty"`
	// CreationDate is the document creation date from PDF InfoDict (PDF only).
	CreationDate *time.Time `json:"creation_date,omitempty"`
	// ModDate is the document modification date from PDF InfoDict (PDF only).
	ModDate *time.Time `json:"mod_date,omitempty"`
	// IsEncrypted reports whether the document is encrypted (PDF only).
	IsEncrypted bool `json:"is_encrypted,omitempty"`
	// EncryptionAlgorithm is the encryption algorithm name when IsEncrypted is true (PDF only).
	EncryptionAlgorithm string `json:"encryption_algorithm,omitempty"`
	// Seals is the list of electronic seal summaries (OFD only).
	Seals []doc.SealSummary `json:"seals,omitempty"`
	// Fonts is the list of font resource summaries (OFD only).
	Fonts []doc.FontSummary `json:"fonts,omitempty"`
	// MediaFiles is the list of multimedia resource summaries (OFD only).
	MediaFiles []doc.MediaSummary `json:"media_files,omitempty"`
	// Pages is the list of per-page metadata including physical dimensions (OFD only).
	Pages []doc.PageInfo `json:"pages,omitempty"`
	// Annotations is the list of per-page annotation summaries (OFD only).
	Annotations []doc.AnnotationSummary `json:"annotations,omitempty"`
}

func runInfoPage(input infoInput, resolver app.ServiceResolver) error {
	format, err := detectFormatByExtension(input.path)
	if err != nil {
		return err
	}

	svc, ok := resolver.ByFormat(format)
	if !ok {
		return fmt.Errorf("no service for format %q", format)
	}

	d, err := svc.Open(context.Background(), doc.DocumentRef{Format: format, Path: input.path})
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
			Path:     input.path,
			MediaBox: result.MediaBox,
		}
		if result.PagesRef.ObjNum != 0 {
			out.PagesRef = pageRef{ObjNum: result.PagesRef.ObjNum, GenNum: result.PagesRef.GenNum}
		}
		if result.PageRef.ObjNum != 0 {
			out.PageRef = pageRef{ObjNum: result.PageRef.ObjNum, GenNum: result.PageRef.GenNum}
		}
		if result.Parent.ObjNum != 0 {
			out.Parent = pageRef{ObjNum: result.Parent.ObjNum, GenNum: result.Parent.GenNum}
		}
		if result.Resources.ObjNum != 0 {
			out.Resources = pageRef{ObjNum: result.Resources.ObjNum, GenNum: result.Resources.GenNum}
		}
		if len(result.Contents) > 0 {
			out.Contents = refsToPageRefs(result.Contents)
		}
		out.Rotate = result.Rotate
		return writeJSON(out)
	}

	fmt.Printf("path: %s\n", input.path)
	fmt.Printf("media_box: %v\n", result.MediaBox)
	if result.PagesRef.ObjNum != 0 || result.PagesRef.GenNum != 0 {
		fmt.Printf("pages_ref: %d %d R\n", result.PagesRef.ObjNum, result.PagesRef.GenNum)
	}
	if result.PageRef.ObjNum != 0 || result.PageRef.GenNum != 0 {
		fmt.Printf("page_ref: %d %d R\n", result.PageRef.ObjNum, result.PageRef.GenNum)
	}
	if result.Parent.ObjNum != 0 || result.Parent.GenNum != 0 {
		fmt.Printf("parent: %d %d R\n", result.Parent.ObjNum, result.Parent.GenNum)
	}
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
