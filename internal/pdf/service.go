package pdf

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PolarKits/polardoc/internal/doc"
)

type service struct{}

type document struct {
	ref             doc.DocumentRef
	file            *os.File
	sizeBytes       int64
	declaredVersion string
}

// NewService returns the PDF service used by phase-1 CLI flows.
func NewService() Service {
	return &service{}
}

func (d *document) Ref() doc.DocumentRef {
	return d.ref
}

func (d *document) Close() error {
	if d.file == nil {
		return nil
	}
	return d.file.Close()
}

func (s *service) Open(_ context.Context, ref doc.DocumentRef) (doc.Document, error) {
	if ref.Format != doc.FormatPDF {
		return nil, fmt.Errorf("format mismatch: expected %q, got %q", doc.FormatPDF, ref.Format)
	}

	f, err := os.Open(ref.Path)
	if err != nil {
		return nil, err
	}

	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	version, err := readPDFHeaderVersion(f)
	if err != nil {
		if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
			f.Close()
			return nil, seekErr
		}
		version = ""
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}

	return &document{
		ref:             ref,
		file:            f,
		sizeBytes:       st.Size(),
		declaredVersion: version,
	}, nil
}

func (s *service) Info(_ context.Context, d doc.Document) (doc.InfoResult, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return doc.InfoResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	return doc.InfoResult{
		Format:          pdfDoc.ref.Format,
		Path:            pdfDoc.ref.Path,
		SizeBytes:       pdfDoc.sizeBytes,
		DeclaredVersion: pdfDoc.declaredVersion,
	}, nil
}

func (s *service) Validate(_ context.Context, d doc.Document) (doc.ValidationReport, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return doc.ValidationReport{}, fmt.Errorf("unsupported document type %T", d)
	}

	if pdfDoc.file == nil {
		return doc.ValidationReport{}, fmt.Errorf("pdf file is not open")
	}

	if _, err := pdfDoc.file.Seek(0, io.SeekStart); err != nil {
		return doc.ValidationReport{}, err
	}

	report := doc.ValidationReport{
		Valid: true,
	}

	if _, err := readPDFHeaderVersion(pdfDoc.file); err != nil {
		report.Valid = false
		report.Errors = append(report.Errors, err.Error())
	}

	if _, err := pdfDoc.file.Seek(0, io.SeekStart); err != nil {
		return doc.ValidationReport{}, err
	}

	return report, nil
}

func (s *service) ExtractText(_ context.Context, _ doc.Document) (doc.TextResult, error) {
	return doc.TextResult{}, nil
}

func (s *service) RenderPreview(_ context.Context, _ doc.Document, _ doc.PreviewRequest) (doc.PreviewResult, error) {
	return doc.PreviewResult{}, fmt.Errorf("preview is not implemented for %q", doc.FormatPDF)
}

func readPDFHeaderVersion(r io.Reader) (string, error) {
	line, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "%PDF-") {
		return "", fmt.Errorf("invalid PDF header")
	}

	return strings.TrimPrefix(line, "%PDF-"), nil
}
