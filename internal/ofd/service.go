package ofd

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PolarKits/polar-doc/internal/doc"
)

// maxZipEntries is the maximum number of files allowed inside an OFD ZIP package.
const maxZipEntries = 10000

// maxDecompressedSize is the maximum total uncompressed size allowed for all
// entries in an OFD ZIP package (512 MiB) to mitigate ZIP bomb attacks.
const maxDecompressedSize = 512 * 1024 * 1024 // 512 MiB

// maxXMLReadSize is the maximum size allowed when reading individual XML files
// from an OFD package (32 MiB) to prevent OOM.
const maxXMLReadSize = 32 * 1024 * 1024 // 32 MiB

// service implements Service with phase-1 OFD capabilities.
type service struct{}

// document holds an opened OFD package handle.
// It maintains the zip reader and metadata needed for Info/Validate operations.
type document struct {
	ref       doc.DocumentRef // original reference used to open the document
	zipReader *zip.ReadCloser // handle to the OFD ZIP package
	sizeBytes int64           // total size of the OFD file in bytes

	metaOnce  sync.Once       // guards lazy initialization of version and pageCount
	version   string          // cached from OFD.xml
	pageCount int             // cached from Document.xml
	metaErr   error           // first error encountered during meta initialization
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

// loadMeta initializes version and pageCount from the OFD package.
// It is called at most once per document via sync.Once.
func (d *document) loadMeta() {
	d.metaOnce.Do(func() {
		v, err := getVersion(d.zipReader.File)
		if err != nil {
			d.metaErr = err
			return
		}
		d.version = v

		pc, err := getPageCount(d.zipReader.File)
		if err != nil {
			d.metaErr = err
			return
		}
		d.pageCount = pc
	})
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

	// Reject ZIP bombs by enforcing entry count and total uncompressed size limits.
	if err := validateZipSafety(zr); err != nil {
		// The validation error is the primary failure reason; close error can be ignored.
		_ = zr.Close()
		return nil, fmt.Errorf("OFD package safety check failed: %w", err)
	}

	return &document{
		ref:       ref,
		zipReader: zr,
		sizeBytes: st.Size(),
	}, nil
}

// Info extracts metadata from an opened OFD document.
// It retrieves format, path, size, version, page count, and seal information
// (if Signatures.xml and Seal.esl files are present).
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

	ofdDoc.loadMeta()
	info.DeclaredVersion = ofdDoc.version
	info.PageCount = ofdDoc.pageCount

	info.Seals = s.collectSealSummaries(ofdDoc.zipReader.File)

	// Parse resource files for font and multimedia metadata.
	if resources, err := ParseResourcesXML(ofdDoc.zipReader.File); err == nil && resources != nil {
		info.Fonts = make([]doc.FontSummary, len(resources.Fonts))
		for i, f := range resources.Fonts {
			info.Fonts[i] = doc.FontSummary{
				FontID:     f.ID,
				FamilyName: f.FamilyName,
				FontName:   f.FontName,
			}
		}
		info.MediaFiles = make([]doc.MediaSummary, len(resources.MultiMedias))
		for i, m := range resources.MultiMedias {
			info.MediaFiles[i] = doc.MediaSummary{
				MediaID: m.ID,
				MediaType: m.Type,
				Format:   m.Format,
			}
		}
	}

	// Collect per-page physical dimensions from Document.xml.
	info.Pages = collectPageInfo(ofdDoc.zipReader.File)

	// Collect per-page annotation metadata from Annotations.xml.
	info.Annotations = collectAnnotationInfo(ofdDoc.zipReader.File)

	return info, nil
}

// collectAnnotationInfo parses Annotations.xml and per-page annotation files
// to build per-page annotation summaries. Returns nil if no annotations found.
func collectAnnotationInfo(files []*zip.File) []doc.AnnotationSummary {
	annotationsDoc, err := ParseAnnotationsXML(files)
	if err != nil || annotationsDoc == nil || len(annotationsDoc.Pages) == 0 {
		return nil
	}

	fileIndex := make(map[string]*zip.File, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	var summaries []doc.AnnotationSummary
	for _, pageIdx := range annotationsDoc.Pages {
		if pageIdx.FilePath == "" {
			continue
		}

		fpath := strings.TrimPrefix(pageIdx.FilePath, "./")
		f, ok := fileIndex[fpath]
		if !ok {
			continue
		}

		data, readErr := readFileContent(f)
		if readErr != nil {
			continue
		}

		annots, parseErr := ParsePageAnnotations(data)
		if parseErr != nil {
			continue
		}

		if len(annots) == 0 {
			continue
		}

		// Deduplicate types.
		typeSet := make(map[string]struct{})
		for _, a := range annots {
			if a.Type != "" {
				typeSet[string(a.Type)] = struct{}{}
			}
		}

		var types []string
		for t := range typeSet {
			types = append(types, t)
		}

		summaries = append(summaries, doc.AnnotationSummary{
			PageID: pageIdx.PageID,
			Count:  len(annots),
			Types:  types,
		})
	}

	if len(summaries) == 0 {
		return nil
	}
	return summaries
}

// collectPageInfo parses Document.xml to extract per-page physical dimensions.
// It first reads the document-level PhysicalBox (from PageArea) as default,
// then iterates over Page elements to get per-page Area/PhysicalBox if present.
func collectPageInfo(files []*zip.File) []doc.PageInfo {
	docData, err := readDocumentXML(files)
	if err != nil {
		return nil
	}

	// Parse document-level PhysicalBox as default dimensions.
	defaultBox := parsePhysicalBox(docData)
	var defaultWidth, defaultHeight float64
	if len(defaultBox) == 4 {
		// MediaBox layout: [llx, lly, urx, ury]; width = urx-llx, height = ury-lly
		defaultWidth = defaultBox[2] - defaultBox[0]
		defaultHeight = defaultBox[3] - defaultBox[1]
	}

	// Parse page list with per-page Area/PhysicalBox.
	pages := parsePageInfoList(docData, defaultWidth, defaultHeight)
	if len(pages) == 0 {
		return nil
	}
	return pages
}

// parsePageInfoList parses Page elements from Document.xml and returns page info.
// Uses document-level default width/height if page has no Area/PhysicalBox.
func parsePageInfoList(docData []byte, defaultWidth, defaultHeight float64) []doc.PageInfo {
	decoder := xml.NewDecoder(bytes.NewReader(docData))
	var pages []doc.PageInfo
	pageNum := 0

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

		pageNum++
		pageInfo := doc.PageInfo{PageNumber: pageNum}

		// Check if Page has Area/PhysicalBox.
		if box := parsePagePhysicalBox(decoder, &se); len(box) == 4 {
			// MediaBox layout: [llx, lly, urx, ury]; width = urx-llx, height = ury-lly
			pageInfo.Width = box[2] - box[0]
			pageInfo.Height = box[3] - box[1]
		} else if defaultWidth > 0 || defaultHeight > 0 {
			pageInfo.Width = defaultWidth
			pageInfo.Height = defaultHeight
		}

		pages = append(pages, pageInfo)
	}

	return pages
}

// parsePagePhysicalBox parses the PhysicalBox from a Page element's Area child.
// It consumes tokens until the Page end element and returns the parsed box.
func parsePagePhysicalBox(decoder *xml.Decoder, pageStart *xml.StartElement) []float64 {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil
		}

		switch v := tok.(type) {
		case xml.EndElement:
			if strings.TrimPrefix(v.Name.Local, "ofd:") == "Page" {
				return nil
			}
		case xml.StartElement:
			local := strings.TrimPrefix(v.Name.Local, "ofd:")
			if local == "Area" {
				if box := parseAreaPhysicalBox(decoder); len(box) == 4 {
					return box
				}
			}
		}
	}
}

// parseAreaPhysicalBox parses PhysicalBox within an Area element.
func parseAreaPhysicalBox(decoder *xml.Decoder) []float64 {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil
		}

		switch v := tok.(type) {
		case xml.EndElement:
			if strings.TrimPrefix(v.Name.Local, "ofd:") == "Area" {
				return nil
			}
		case xml.StartElement:
			local := strings.TrimPrefix(v.Name.Local, "ofd:")
			if local == "PhysicalBox" {
				var content string
				if err := decoder.DecodeElement(&content, &v); err == nil {
					return parsePhysicalBox([]byte(content))
				}
			}
		}
	}
}

// collectSealSummaries parses Signatures.xml and associated Seal.esl files
// to build a list of SealSummary entries. Returns nil if no signatures found.
func (s *service) collectSealSummaries(files []*zip.File) []doc.SealSummary {
	sigs, err := ParseSignaturesXML(files)
	if err != nil || len(sigs) == 0 {
		return nil
	}

	var summaries []doc.SealSummary
	for _, sig := range sigs {
		if sig.SealBaseLoc == "" {
			continue
		}

		summary := doc.SealSummary{ID: sig.ID}
		loc := strings.TrimPrefix(sig.SealBaseLoc, "./")
		loc = strings.TrimPrefix(loc, "/") // absolute path from OFD may start with "/"

		for _, f := range files {
			name := strings.TrimPrefix(f.Name, "./")
			if name == loc {
				rc, err := f.Open()
				if err != nil {
					break
				}
				data, err := io.ReadAll(rc)
				rc.Close()
				if err != nil {
					break
				}
				result, parseErr := ParseSealESL(data)
				if parseErr == nil {
					summary.Version = result.Seal.Version
					summary.Width = result.Seal.Width
					summary.Height = result.Seal.Height
					summary.PictureFormat = result.Seal.Picture.Format
				}
				break
			}
		}

		summaries = append(summaries, summary)
	}

	if len(summaries) == 0 {
		return nil
	}
	return summaries
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

	// Check seal structure integrity if Signatures.xml exists.
	for _, warnText := range s.validateSeals(ofdDoc.zipReader.File) {
		report.Warnings = append(report.Warnings, warnText)
	}

	// Check resource file references if Resources.xml exists.
	for _, warnText := range s.validateResources(ofdDoc.zipReader.File) {
		report.Warnings = append(report.Warnings, warnText)
	}

	return report, nil
}

// validateResources checks that font and multimedia files referenced in
// Resources.xml actually exist in the OFD package. Missing files are
// added as warnings, not errors.
func (s *service) validateResources(files []*zip.File) []string {
	resources, err := ParseResourcesXML(files)
	if err != nil || resources == nil {
		return nil
	}

	var warnings []string
	fileIndex := make(map[string]*zip.File, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	for _, font := range resources.Fonts {
		if font.FilePath == "" {
			continue
		}
		fpath := strings.TrimPrefix(font.FilePath, "./")
		fpath = strings.TrimPrefix(fpath, "/") // absolute path from OFD may start with "/"
		if _, ok := fileIndex[fpath]; !ok {
			warnings = append(warnings, fmt.Sprintf("font %d: file not found: %s", font.ID, font.FilePath))
		}
	}

	for _, mm := range resources.MultiMedias {
		if mm.FilePath == "" {
			continue
		}
		fpath := strings.TrimPrefix(mm.FilePath, "./")
		fpath = strings.TrimPrefix(fpath, "/") // absolute path from OFD may start with "/"
		if _, ok := fileIndex[fpath]; !ok {
			warnings = append(warnings, fmt.Sprintf("multimedia %d: file not found: %s", mm.ID, mm.FilePath))
		}
	}

	return warnings
}

// validateSeals checks signature seal structure integrity.
// It verifies each signature's SealBaseLoc points to an existing Seal.esl
// file that can be parsed. Issues are added as warnings, not errors.
func (s *service) validateSeals(files []*zip.File) []string {
	sigs, err := ParseSignaturesXML(files)
	if err != nil || len(sigs) == 0 {
		return nil
	}

	var warnings []string
	fileIndex := make(map[string]*zip.File, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	for _, sig := range sigs {
		if sig.SealBaseLoc == "" {
			warnings = append(warnings, fmt.Sprintf("signature %d: missing SealBaseLoc", sig.ID))
			continue
		}
		loc := strings.TrimPrefix(sig.SealBaseLoc, "./")
		loc = strings.TrimPrefix(loc, "/") // absolute path from OFD may start with "/"

		f, ok := fileIndex[loc]
		if !ok {
			warnings = append(warnings, fmt.Sprintf("signature %d: Seal.esl not found at %s", sig.ID, sig.SealBaseLoc))
			continue
		}

		data, readErr := readSealFile(f)
		if readErr != nil {
			warnings = append(warnings, fmt.Sprintf("signature %d: failed to read Seal.esl: %v", sig.ID, readErr))
			continue
		}

		_, parseErr := ParseSealESL(data)
		if parseErr != nil {
			warnings = append(warnings, fmt.Sprintf("signature %d: Seal.esl parse failed: %v", sig.ID, parseErr))
		}
	}

	return warnings
}

// readSealFile reads and returns the content of a seal file.
func readSealFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
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

// ExtractTextPage extracts text from a single page (1-based page number).
// Implements doc.PagedTextExtractor.
func (s *service) ExtractTextPage(_ context.Context, d doc.Document, pageNum int) (doc.TextResult, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return doc.TextResult{}, fmt.Errorf("unsupported document type %T", d)
	}
	if ofdDoc.zipReader == nil {
		return doc.TextResult{}, fmt.Errorf("OFD package is not open")
	}

	text, err := extractOFDTextPage(ofdDoc.zipReader.File, pageNum)
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
	docData, err := readLimited(rc, maxXMLReadSize, "Document.xml")
	if err != nil {
		return "", fmt.Errorf("extractOFDText: %w", err)
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
		pageData, err := readLimited(cr, maxXMLReadSize, "Content.xml")
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

// extractOFDTextPage extracts text from a single page (1-based page number).
// It follows the same file lookup and Document.xml parsing logic as extractOFDText.
func extractOFDTextPage(files []*zip.File, pageNum int) (string, error) {
	if pageNum < 1 {
		return "", fmt.Errorf("page %d out of range", pageNum)
	}

	fileIndex := make(map[string]*zip.File, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	docRoot, err := getDocRoot(files)
	if err != nil {
		return "", fmt.Errorf("extractOFDTextPage: %w", err)
	}
	docRoot = strings.TrimPrefix(docRoot, "./")

	docFile, ok := fileIndex[docRoot]
	if !ok {
		return "", fmt.Errorf("extractOFDTextPage: Document.xml not found at %q", docRoot)
	}

	rc, err := docFile.Open()
	if err != nil {
		return "", fmt.Errorf("extractOFDTextPage: open Document.xml: %w", err)
	}
	docData, err := readLimited(rc, maxXMLReadSize, "Document.xml")
	if err != nil {
		return "", fmt.Errorf("extractOFDTextPage: %w", err)
	}
	_ = rc.Close()

	docDir := ""
	if slash := strings.LastIndex(docRoot, "/"); slash >= 0 {
		docDir = docRoot[:slash+1]
	}

	pageLocations := ofdPageLocations(docData)
	if pageNum > len(pageLocations) {
		return "", fmt.Errorf("page %d out of range (document has %d pages)", pageNum, len(pageLocations))
	}

	relLoc := pageLocations[pageNum-1]
	absLoc := docDir + strings.TrimPrefix(relLoc, "./")
	contentFile, found := fileIndex[absLoc]
	if !found {
		return "", fmt.Errorf("extractOFDTextPage: Content.xml not found at %q", absLoc)
	}

	cr, err := contentFile.Open()
	if err != nil {
		return "", fmt.Errorf("extractOFDTextPage: open Content.xml: %w", err)
	}
	pageData, err := readLimited(cr, maxXMLReadSize, "Content.xml")
	if err != nil {
		_ = cr.Close()
		return "", fmt.Errorf("extractOFDTextPage: %w", err)
	}
	_ = cr.Close()

	return extractTextCodesFromPage(pageData), nil
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

// FirstPageInfo returns first page information for OFD documents.
// It extracts the PhysicalBox from Document.xml's PageArea as MediaBox.
// Returns (nil, nil) if PhysicalBox is not present; returns error only on parse failure.
func (s *service) FirstPageInfo(_ context.Context, d doc.Document) (*doc.FirstPageInfoResult, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return nil, fmt.Errorf("unsupported document type %T", d)
	}

	docData, err := readDocumentXML(ofdDoc.zipReader.File)
	if err != nil {
		return nil, fmt.Errorf("read Document.xml: %w", err)
	}

	mediaBox := parsePhysicalBox(docData)

	result := &doc.FirstPageInfoResult{
		Path:     ofdDoc.ref.Path,
		MediaBox: mediaBox,
	}
	return result, nil
}

// readDocumentXML reads the Document.xml file from the OFD package.
// It returns the raw XML content.
func readDocumentXML(files []*zip.File) ([]byte, error) {
	fileIndex := make(map[string]*zip.File, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	docRoot, err := getDocRoot(files)
	if err != nil {
		return nil, fmt.Errorf("get DocRoot: %w", err)
	}
	docRoot = strings.TrimPrefix(docRoot, "./")

	docFile, ok := fileIndex[docRoot]
	if !ok {
		return nil, fmt.Errorf("Document.xml not found at %q", docRoot)
	}

	rc, err := docFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open Document.xml: %w", err)
	}
	return readLimited(rc, maxXMLReadSize, "Document.xml")
}

// parsePhysicalBox parses the PhysicalBox element from Document.xml data.
// Returns nil if PhysicalBox is not found.
// PhysicalBox format: "x y width height" → []float64{x, y, x+width, y+height} (MediaBox format).
func parsePhysicalBox(data []byte) []float64 {
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
		if se.Name.Local == "PhysicalBox" {
			var content string
			if err := decoder.DecodeElement(&content, &se); err != nil {
				return nil
			}
			content = strings.TrimSpace(content)
			if content == "" {
				return nil
			}
			parts := strings.Fields(content)
			if len(parts) != 4 {
				return nil
			}
			var vals [4]float64
			for i := 0; i < 4; i++ {
				v, err := strconv.ParseFloat(parts[i], 64)
				if err != nil {
					return nil
				}
				vals[i] = v
			}
			// Convert to MediaBox format [llx, lly, urx, ury]
			// PhysicalBox is "x y width height", MediaBox is "x1 y1 x2 y2"
			return []float64{vals[0], vals[1], vals[0] + vals[2], vals[1] + vals[3]}
		}
	}
	return nil
}

// ofdPageIterator provides sequential streaming access to OFD pages.
// It uses the page location list derived from Document.xml.
type ofdPageIterator struct {
	doc       *document
	locations []string
	index     int
	fileIndex map[string]*zip.File
}

// newOFDPageIterator creates a new OFD page iterator for the given document.
func newOFDPageIterator(d *document) (*ofdPageIterator, error) {
	fileIndex := make(map[string]*zip.File, len(d.zipReader.File))
	for _, f := range d.zipReader.File {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	docRoot, err := getDocRoot(d.zipReader.File)
	if err != nil {
		return nil, fmt.Errorf("newOFDPageIterator: %w", err)
	}
	docRoot = strings.TrimPrefix(docRoot, "./")

	docFile, ok := fileIndex[docRoot]
	if !ok {
		return nil, fmt.Errorf("newOFDPageIterator: Document.xml not found at %q", docRoot)
	}

	rc, err := docFile.Open()
	if err != nil {
		return nil, fmt.Errorf("newOFDPageIterator: open Document.xml: %w", err)
	}
	docData, err := readLimited(rc, maxXMLReadSize, "Document.xml")
	if err != nil {
		return nil, fmt.Errorf("newOFDPageIterator: %w", err)
	}

	docDir := ""
	if slash := strings.LastIndex(docRoot, "/"); slash >= 0 {
		docDir = docRoot[:slash+1]
	}

	relLocations := ofdPageLocations(docData)
	locations := make([]string, len(relLocations))
	for i, relLoc := range relLocations {
		locations[i] = docDir + strings.TrimPrefix(relLoc, "./")
	}

	return &ofdPageIterator{
		doc:       d,
		locations: locations,
		index:     0,
		fileIndex: fileIndex,
	}, nil
}

// Next returns the next page data or io.EOF when exhausted.
func (it *ofdPageIterator) Next(ctx context.Context) (doc.PageData, error) {
	if it.index >= len(it.locations) {
		return doc.PageData{}, io.EOF
	}
	absLoc := it.locations[it.index]
	contentFile, ok := it.fileIndex[absLoc]
	if !ok {
		return doc.PageData{}, fmt.Errorf("Next: Content.xml not found at %q", absLoc)
	}
	rc, err := contentFile.Open()
	if err != nil {
		return doc.PageData{}, fmt.Errorf("Next: open Content.xml: %w", err)
	}
	data, err := readLimited(rc, maxXMLReadSize, "Content.xml")
	if err != nil {
		return doc.PageData{}, fmt.Errorf("Next: read Content.xml: %w", err)
	}
	pageNum := it.index + 1
	it.index++
	return doc.PageData{Number: pageNum, ObjRef: absLoc, MediaBox: nil, Content: data}, nil
}

// Reset restarts the iterator from the first page.
func (it *ofdPageIterator) Reset() {
	it.index = 0
}

// ofdNavigator provides random-access to OFD page content via Content.xml paths.
type ofdNavigator struct {
	doc       *document
	fileIndex map[string]*zip.File
	locations []string
}

// GoTo resolves a page object reference and returns its content.
// The ref is an absolute zip path to a Content.xml file.
func (nav *ofdNavigator) GoTo(ctx context.Context, ref string) (doc.PageData, error) {
	contentFile, ok := nav.fileIndex[ref]
	if !ok {
		return doc.PageData{}, fmt.Errorf("GoTo: Content.xml not found at %q", ref)
	}
	rc, err := contentFile.Open()
	if err != nil {
		return doc.PageData{}, fmt.Errorf("GoTo: open Content.xml: %w", err)
	}
	data, err := readLimited(rc, maxXMLReadSize, "Content.xml")
	if err != nil {
		return doc.PageData{}, fmt.Errorf("GoTo: read Content.xml: %w", err)
	}
	pageNum := 0
	for i, loc := range nav.locations {
		if loc == ref {
			pageNum = i + 1
			break
		}
	}
	// pageNum remains 0 if ref was not found in locations (unknown page).
	return doc.PageData{Number: pageNum, ObjRef: ref, MediaBox: nil, Content: data}, nil
}

// NewPageIterator implements doc.PageIteratorProvider.
func (s *service) NewPageIterator(ctx context.Context, d doc.Document) (doc.PageIterator, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return nil, fmt.Errorf("NewPageIterator: unsupported document type %T", d)
	}
	return newOFDPageIterator(ofdDoc)
}

// NewNavigator implements doc.NavigatorProvider.
func (s *service) NewNavigator(ctx context.Context, d doc.Document) (doc.Navigator, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return nil, fmt.Errorf("NewNavigator: unsupported document type %T", d)
	}
	iter, err := newOFDPageIterator(ofdDoc)
	if err != nil {
		return nil, err
	}
	return &ofdNavigator{doc: ofdDoc, fileIndex: iter.fileIndex, locations: iter.locations}, nil
}

func (s *service) Sign(_ context.Context, d doc.Document, _ doc.SignRequest) (doc.SignResult, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return doc.SignResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	_ = ofdDoc
	return doc.SignResult{}, fmt.Errorf("signing is not implemented for %q", doc.FormatOFD)
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
			data, err := readLimited(rc, maxXMLReadSize, "OFD.xml")
			if err != nil {
				return "", err
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
			data, err := readLimited(rc, maxXMLReadSize, "OFD.xml")
			if err != nil {
				return "", err
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
	data, err := readLimited(rc, maxXMLReadSize, "Document.xml")
	if err != nil {
		return 0, err
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

// validateZipSafety checks the ZIP file for potential decompression bombs.
// It limits the number of entries and the total uncompressed size.
func validateZipSafety(zr *zip.ReadCloser) error {
	if len(zr.File) > maxZipEntries {
		return fmt.Errorf("ZIP file contains %d entries, exceeding the maximum allowed %d", len(zr.File), maxZipEntries)
	}

	var totalUncompressed int64
	for _, f := range zr.File {
		totalUncompressed += int64(f.UncompressedSize64)
		if totalUncompressed > maxDecompressedSize {
			return fmt.Errorf("total uncompressed size exceeds maximum allowed %d bytes", maxDecompressedSize)
		}
	}
	return nil
}

// readLimited reads all data from rc with a size limit to prevent OOM attacks.
// It closes rc before returning. If the data reaches the limit, it returns an error
// indicating the content was truncated.
func readLimited(rc io.ReadCloser, limit int64, name string) ([]byte, error) {
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, limit))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", name, err)
	}
	if int64(len(data)) >= limit {
		return nil, fmt.Errorf("%s exceeds maximum allowed size (%d bytes) and was truncated", name, limit)
	}
	return data, nil
}
