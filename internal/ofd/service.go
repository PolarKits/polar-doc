package ofd

import (
	"archive/zip"
	"context"
	"encoding/xml"
	"fmt"
	"io"
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

	entryErrs := validateOFDEntries(ofdDoc.zipReader.File)
	for _, errText := range entryErrs {
		report.Valid = false
		report.Errors = append(report.Errors, errText)
	}

	hasDocument := false
	for _, f := range ofdDoc.zipReader.File {
		name := strings.TrimPrefix(f.Name, "./")
		if strings.HasSuffix(name, "/Document.xml") {
			hasDocument = true
			break
		}
	}

	if !hasDocument {
		return report, nil
	}

	docRoot, err := getDocRoot(ofdDoc.zipReader.File)
	if err != nil {
		report.Valid = false
		report.Errors = append(report.Errors, fmt.Sprintf("failed to parse DocRoot: %v", err))
	} else {
		for _, errText := range validateDocRoot(ofdDoc.zipReader.File, docRoot) {
			report.Valid = false
			report.Errors = append(report.Errors, errText)
		}
	}

	return report, nil
}

func (s *service) ExtractText(_ context.Context, d doc.Document) (doc.TextResult, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return doc.TextResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	_ = ofdDoc
	return doc.TextResult{}, fmt.Errorf("text extraction is not implemented for OFD")
}

func (s *service) RenderPreview(_ context.Context, d doc.Document, _ doc.PreviewRequest) (doc.PreviewResult, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return doc.PreviewResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	_ = ofdDoc
	return doc.PreviewResult{}, fmt.Errorf("preview is not implemented for %q", doc.FormatOFD)
}

func (s *service) FirstPageInfo(_ context.Context, d doc.Document) (*doc.FirstPageInfoResult, error) {
	_, ok := d.(*document)
	if !ok {
		return nil, fmt.Errorf("unsupported document type %T", d)
	}
	return nil, fmt.Errorf("first page info not supported for OFD")
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

func getDocRoot(files []*zip.File) (string, error) {
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		if name == "OFD.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open OFD.xml: %w", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("failed to read OFD.xml: %w", err)
			}

			type ofdXML struct {
				DocRoot string `xml:"DocRoot"`
			}
			var parsed ofdXML
			if err := xml.Unmarshal(data, &parsed); err != nil {
				return "", fmt.Errorf("failed to parse OFD.xml: %w", err)
			}
			return strings.TrimSpace(parsed.DocRoot), nil
		}
	}
	return "", fmt.Errorf("OFD.xml not found")
}

func validateDocRoot(files []*zip.File, docRoot string) []string {
	if docRoot == "" {
		return []string{"DocRoot element is missing or empty in OFD.xml"}
	}

	normalizedDocRoot := strings.TrimPrefix(strings.TrimPrefix(docRoot, "./"), "./")
	docRootFound := false

	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		if name == normalizedDocRoot {
			docRootFound = true
			break
		}
	}

	if !docRootFound {
		return []string{fmt.Sprintf("DocRoot %q points to a non-existent file", docRoot)}
	}

	return nil
}
