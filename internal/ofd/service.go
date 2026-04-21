package ofd

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/PolarKits/polardoc/internal/doc"
)

// service implements Service with phase-1 OFD capabilities.
type service struct{}

// document holds an opened OFD package handle.
// It maintains the zip reader and metadata needed for Info/Validate operations.
type document struct {
	ref       doc.DocumentRef // original reference used to open the document
	zipReader *zip.ReadCloser // handle to the OFD ZIP package
	sizeBytes int64           // total size of the OFD file in bytes
}

// NewService returns the OFD service used by phase-1 CLI flows.
func NewService() Service {
	return &service{}
}

// Ref returns the document reference used to open this document.
func (d *document) Ref() doc.DocumentRef {
	return d.ref
}

// Close releases the OFD package handle.
// It is safe to call Close multiple times; subsequent calls are no-ops.
func (d *document) Close() error {
	if d.zipReader == nil {
		return nil
	}
	return d.zipReader.Close()
}

// Open opens an OFD package from the given reference.
// It validates the format matches OFD and returns a document handle.
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

// Info extracts metadata from an opened OFD document.
// It retrieves format, path, size, version, and page count (if Document.xml is present).
func (s *service) Info(_ context.Context, d doc.Document) (doc.InfoResult, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return doc.InfoResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	info := doc.InfoResult{
		Format:    ofdDoc.ref.Format,
		Path:      ofdDoc.ref.Path,
		SizeBytes: ofdDoc.sizeBytes,
	}

	version, _ := getVersion(ofdDoc.zipReader.File)
	info.DeclaredVersion = version

	pageCount, _ := getPageCount(ofdDoc.zipReader.File)
	info.PageCount = pageCount

	return info, nil
}

// Validate checks the OFD package structure and returns a validation report.
// It verifies OFD.xml and Document.xml presence, and validates DocRoot references.
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

// ExtractText extracts text content from all pages in the OFD document.
// It walks through each page's Content.xml and collects TextCode elements.
func (s *service) ExtractText(_ context.Context, d doc.Document) (doc.TextResult, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return doc.TextResult{}, fmt.Errorf("unsupported document type %T", d)
	}
	if ofdDoc.zipReader == nil {
		return doc.TextResult{}, fmt.Errorf("OFD package is not open")
	}

	text, err := extractOFDText(ofdDoc.zipReader.File)
	if err != nil {
		return doc.TextResult{}, err
	}
	return doc.TextResult{Text: strings.TrimSpace(text)}, nil
}

// extractOFDText reads all page Content.xml files and collects TextCode text.
// It follows the Document.xml page list to enumerate pages in order, then
// scans each page's Content.xml for TextObject/TextCode elements.
func extractOFDText(files []*zip.File) (string, error) {
	// Build a filename → zip.File index for O(1) lookup.
	fileIndex := make(map[string]*zip.File, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	// Locate Document.xml via OFD.xml's DocRoot.
	docRoot, err := getDocRoot(files)
	if err != nil {
		return "", fmt.Errorf("extractOFDText: %w", err)
	}
	docRoot = strings.TrimPrefix(docRoot, "./")

	docFile, ok := fileIndex[docRoot]
	if !ok {
		return "", fmt.Errorf("extractOFDText: Document.xml not found at %q", docRoot)
	}

	// Parse Document.xml to find all <Page BaseLoc="..."/> entries.
	rc, err := docFile.Open()
	if err != nil {
		return "", fmt.Errorf("extractOFDText: open Document.xml: %w", err)
	}
	docData, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return "", fmt.Errorf("extractOFDText: read Document.xml: %w", err)
	}

	// docRoot's directory is the base for relative page paths.
	docDir := ""
	if slash := strings.LastIndex(docRoot, "/"); slash >= 0 {
		docDir = docRoot[:slash+1]
	}

	pageLocations := ofdPageLocations(docData)

	// Extract text from each page's Content.xml.
	var sb strings.Builder
	for _, relLoc := range pageLocations {
		// Page BaseLoc is relative to Document.xml's directory.
		absLoc := docDir + strings.TrimPrefix(relLoc, "./")
		contentFile, found := fileIndex[absLoc]
		if !found {
			continue
		}
		cr, err := contentFile.Open()
		if err != nil {
			continue
		}
		pageData, err := io.ReadAll(cr)
		cr.Close()
		if err != nil {
			continue
		}
		pageText := extractTextCodesFromPage(pageData)
		if sb.Len() > 0 && len(pageText) > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(pageText)
	}

	return sb.String(), nil
}

// ofdPageLocations parses a Document.xml byte slice and returns the BaseLoc
// attribute values of all <ofd:Page> elements in document order.
func ofdPageLocations(data []byte) []string {
	var locs []string
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		local := strings.TrimPrefix(se.Name.Local, "ofd:")
		if local != "Page" {
			continue
		}
		for _, attr := range se.Attr {
			if attr.Name.Local == "BaseLoc" {
				locs = append(locs, attr.Value)
				break
			}
		}
	}
	return locs
}

// extractTextCodesFromPage parses a Content.xml byte slice and returns all
// TextCode element text values joined by spaces.
func extractTextCodesFromPage(data []byte) string {
	var parts []string
	decoder := xml.NewDecoder(bytes.NewReader(data))
	inTextCode := false
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			local := strings.TrimPrefix(t.Name.Local, "ofd:")
			if local == "TextCode" {
				inTextCode = true
			}
		case xml.EndElement:
			local := strings.TrimPrefix(t.Name.Local, "ofd:")
			if local == "TextCode" {
				inTextCode = false
			}
		case xml.CharData:
			if inTextCode {
				text := strings.TrimSpace(string(t))
				if text != "" {
					parts = append(parts, text)
				}
			}
		}
	}
	return strings.Join(parts, " ")
}

// RenderPreview returns an error indicating preview is not implemented for OFD.
// This is a phase-1 limitation; full preview rendering is planned for future phases.
func (s *service) RenderPreview(_ context.Context, d doc.Document, _ doc.PreviewRequest) (doc.PreviewResult, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return doc.PreviewResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	_ = ofdDoc
	return doc.PreviewResult{}, fmt.Errorf("preview is not implemented for %q", doc.FormatOFD)
}

// FirstPageInfo returns an error indicating first page info is not supported for OFD.
// This operation is intentionally not implemented for OFD format.
func (s *service) FirstPageInfo(_ context.Context, d doc.Document) (*doc.FirstPageInfoResult, error) {
	_, ok := d.(*document)
	if !ok {
		return nil, fmt.Errorf("unsupported document type %T", d)
	}
	return nil, fmt.Errorf("first page info not supported for OFD")
}

// validateOFDEntries checks for mandatory OFD.xml and Document.xml entries.
// Returns a list of validation error messages for missing entries.
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

// getDocRoot extracts the DocRoot path from OFD.xml.
// It locates the OFD.xml entry and parses the DocRoot element value.
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

			// Use streaming decoder to find DocRoot regardless of nesting depth.
			// Real OFD files nest DocRoot inside DocBody; simplified test fixtures
			// may place it as a direct child of the root element.
			decoder := xml.NewDecoder(bytes.NewReader(data))
			for {
				tok, decErr := decoder.Token()
				if decErr != nil {
					break
				}
				se, ok := tok.(xml.StartElement)
				if !ok {
					continue
				}
				local := se.Name.Local
				if local == "DocRoot" {
					var content string
					if err := decoder.DecodeElement(&content, &se); err == nil {
						if v := strings.TrimSpace(content); v != "" {
							return v, nil
						}
					}
				}
			}
			return "", nil
		}
	}
	return "", fmt.Errorf("OFD.xml not found")
}

// getVersion extracts the Version from OFD.xml.
// It checks both Version attribute (real OFD format) and Version element (simplified test format).
func getVersion(files []*zip.File) (string, error) {
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

			// Use streaming decoder to handle both:
			//   - Real OFD files: version in <ofd:OFD Version="1.0"> attribute
			//   - Simplified test files: version in <Version>1.0</Version> child element
			decoder := xml.NewDecoder(bytes.NewReader(data))
			for {
				tok, decErr := decoder.Token()
				if decErr != nil {
					break
				}
				se, ok := tok.(xml.StartElement)
				if !ok {
					continue
				}
				local := se.Name.Local
				// Check for Version attribute on any root-level element (e.g. <ofd:OFD Version="1.0">).
				for _, attr := range se.Attr {
					if attr.Name.Local == "Version" {
						if v := strings.TrimSpace(attr.Value); v != "" {
							return v, nil
						}
					}
				}
				// Also check for Version as a child element (simplified test format).
				if local == "Version" {
					var content string
					if err := decoder.DecodeElement(&content, &se); err == nil {
						if v := strings.TrimSpace(content); v != "" {
							return v, nil
						}
					}
				}
			}
			return "", nil
		}
	}
	return "", fmt.Errorf("OFD.xml not found")
}

// validateDocRoot verifies that the DocRoot path points to an existing file in the package.
// Returns validation errors if DocRoot is empty or points to a non-existent file.
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

// getPageCount counts the number of Page elements in Document.xml.
// Returns 0 if Document.xml cannot be located or parsed (graceful degradation).
func getPageCount(files []*zip.File) (int, error) {
	docRoot, err := getDocRoot(files)
	if err != nil {
		return 0, err
	}

	normalizedDocRoot := strings.TrimPrefix(strings.TrimPrefix(docRoot, "./"), "./")

	var docFile *zip.File
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		if name == normalizedDocRoot {
			docFile = f
			break
		}
	}

	if docFile == nil {
		return 0, fmt.Errorf("Document.xml not found at %s", docRoot)
	}

	rc, err := docFile.Open()
	if err != nil {
		return 0, fmt.Errorf("failed to open Document.xml: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return 0, fmt.Errorf("failed to read Document.xml: %w", err)
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	pageCount := 0
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to parse Document.xml: %w", err)
		}

		if se, ok := token.(xml.StartElement); ok {
			localName := strings.TrimPrefix(se.Name.Local, "ofd:")
			if localName == "Page" {
				pageCount++
			}
		}
	}

	return pageCount, nil
}
