package ofd

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/PolarKits/polardoc/internal/doc"
)

type service struct{}

type document struct {
	ref       doc.DocumentRef
	zipReader *zip.ReadCloser
	sizeBytes int64
}

// NewService returns the OFD service used by phase-1 CLI flows.
func NewService() Service {
	return &service{}
}

func (d *document) Ref() doc.DocumentRef {
	return d.ref
}

func (d *document) Close() error {
	if d.zipReader == nil {
		return nil
	}
	return d.zipReader.Close()
}

func (s *service) Open(_ context.Context, ref doc.DocumentRef) (doc.Document, error) {
	if ref.Format != doc.FormatOFD {
		return nil, fmt.Errorf("format mismatch: expected %q, got %q", doc.FormatOFD, ref.Format)
	}

	st, err := os.Stat(ref.Path)
	if err != nil {
		return nil, err
	}

	zr, err := zip.OpenReader(ref.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open OFD package: %w", err)
	}

	return &document{
		ref:       ref,
		zipReader: zr,
		sizeBytes: st.Size(),
	}, nil
}

func (s *service) Info(_ context.Context, d doc.Document) (doc.InfoResult, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return doc.InfoResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	return doc.InfoResult{
		Format:    ofdDoc.ref.Format,
		Path:      ofdDoc.ref.Path,
		SizeBytes: ofdDoc.sizeBytes,
	}, nil
}

func (s *service) Validate(_ context.Context, d doc.Document) (doc.ValidationReport, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return doc.ValidationReport{}, fmt.Errorf("unsupported document type %T", d)
	}

	if ofdDoc.zipReader == nil {
		return doc.ValidationReport{}, fmt.Errorf("ofd package is not open")
	}

	report := doc.ValidationReport{
		Valid: true,
	}

	for _, errText := range validateOFDEntries(ofdDoc.zipReader.File) {
		report.Valid = false
		report.Errors = append(report.Errors, errText)
	}

	return report, nil
}

func (s *service) ExtractText(_ context.Context, _ doc.Document) (doc.TextResult, error) {
	return doc.TextResult{}, nil
}

func (s *service) RenderPreview(_ context.Context, _ doc.Document, _ doc.PreviewRequest) (doc.PreviewResult, error) {
	return doc.PreviewResult{}, fmt.Errorf("preview is not implemented for %q", doc.FormatOFD)
}

func validateOFDEntries(files []*zip.File) []string {
	hasRoot := false
	hasDocument := false

	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		if name == "OFD.xml" {
			hasRoot = true
		}
		if strings.HasSuffix(name, "/Document.xml") {
			hasDocument = true
		}
	}

	var errs []string
	if !hasRoot {
		errs = append(errs, "invalid OFD package: missing OFD.xml")
	}
	if !hasDocument {
		errs = append(errs, "invalid OFD package: missing Document.xml")
	}

	return errs
}
