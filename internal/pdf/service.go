package pdf

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PolarKits/polar-doc/internal/doc"
)

// FirstPageInfo holds raw PDF object references and attributes for the first page.
//
// This structure captures the structural information needed to render or extract
// content from the first page of a PDF document, including:
//   - References to the Pages tree and the specific Page object
//   - MediaBox (page dimensions) in PDF user units
//   - Resources dictionary (fonts, images, etc.) reference
//   - Content stream references for page drawing commands
//
// All coordinates and dimensions are in the PDF coordinate system (points,
// origin at bottom-left unless rotated).
type FirstPageInfo struct {
	// PagesRef is the indirect reference to the root Pages tree object.
	PagesRef PDFRef
	// PageRef is the indirect reference to the first Page object.
	PageRef PDFRef
	// Parent is the indirect reference to the immediate parent (Pages node).
	Parent PDFRef
	// MediaBox is the page bounding box as a PDF array [x y width height].
	MediaBox PDFArray
	// Resources is the indirect reference to the Resources dictionary (if external).
	Resources PDFRef
	// InlineResources is the inline Resources dictionary (if present directly in the Page).
	InlineResources PDFDict
	// Contents is the list of indirect references to the page's content stream(s).
	Contents []PDFRef
	// Rotate is the page rotation angle in degrees (0, 90, 180, or 270).
	// Nil means no rotation is specified; callers should default to 0.
	Rotate *int64
}

const (
	// maxPageTreeDepth limits recursion when traversing the PDF Pages tree
	// to prevent stack overflow from maliciously deep or cyclic trees.
	maxPageTreeDepth = 64
	// maxAncestorLookupDepth limits recursion when looking up inherited
	// page attributes (MediaBox, Resources, Rotate) through ancestor Pages.
	maxAncestorLookupDepth = 64
	// maxStreamLength caps the size of a single content stream buffer to
	// prevent OOM when a PDF declares an abnormally large /Length.
	maxStreamLength = 64 * 1024 * 1024 // 64 MiB
)

type service struct{}

type document struct {
	ref             doc.DocumentRef
	file            *os.File
	sizeBytes       int64
	declaredVersion string
	xrefIdx         xrefIndex          // unified xref index (lazy-loaded on first access)
	xrefStartOffset int64              // byte offset of the newest xref section
	xrefOffsets     []int64            // ordered chain of xref section offsets (newest first)
	xrefLoaded      map[int64]bool     // tracks which sections have been parsed into xrefIdx
	mu              sync.Mutex         // guards xrefIdx, xrefOffsets, xrefLoaded
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

func (d *document) getFile() *os.File {
	return d.file
}

// getXRefIndex returns the unified xref index for this document.
// The index is lazily loaded on first access and cached for subsequent calls.
// This method is safe for concurrent use.
func (d *document) getXRefIndex() (xrefIndex, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.xrefIdx != nil {
		return d.xrefIdx, nil
	}

	if d.xrefStartOffset == 0 {
		xrefOffset, err := readStartxref(d.file)
		if err != nil {
			return nil, err
		}
		d.xrefStartOffset = xrefOffset
	}

	idx, err := buildXRefIndex(d.file, d.xrefStartOffset)
	if err != nil {
		return nil, err
	}
	d.xrefIdx = idx
	return idx, nil
}

// getXRefEntry returns the xref entry for the given object number.
// Fast path: entry is already cached in the index.
// Slow path: xref sections are loaded one at a time (newest first) until the
// target object is found or all sections are exhausted.
// This method is safe for concurrent use.
func (d *document) getXRefEntry(objNum int64) (xrefEntry, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Fast path: already cached
	if d.xrefIdx != nil {
		if entry, ok := d.xrefIdx[objNum]; ok {
			return entry, nil
		}
		// All sections already loaded — object genuinely absent
		if d.xrefLoaded != nil && len(d.xrefLoaded) == len(d.xrefOffsets) {
			return xrefEntry{}, fmt.Errorf("object %d not found in xref", objNum)
		}
	}

	// Ensure offset chain is discovered
	if d.xrefOffsets == nil {
		if d.xrefStartOffset == 0 {
			xrefOffset, err := readStartxref(d.file)
			if err != nil {
				return xrefEntry{}, err
			}
			d.xrefStartOffset = xrefOffset
		}
		offsets, err := discoverXRefOffsets(d.file, d.xrefStartOffset)
		if err != nil {
			return xrefEntry{}, err
		}
		d.xrefOffsets = offsets
		d.xrefLoaded = make(map[int64]bool)
		if d.xrefIdx == nil {
			d.xrefIdx = make(xrefIndex)
		}
	}

	// Slow path: load sections one by one until the object is found
	for _, offset := range d.xrefOffsets {
		if d.xrefLoaded[offset] {
			continue
		}

		_, entries, objNums, err := parseXRefSectionAt(d.file, offset)
		if err != nil {
			return xrefEntry{}, fmt.Errorf("lazy-load xref section at %d: %w", offset, err)
		}
		d.xrefLoaded[offset] = true

		for i, entry := range entries {
			if i < len(objNums) {
				if _, exists := d.xrefIdx[objNums[i]]; !exists {
					d.xrefIdx[objNums[i]] = entry
				}
			}
		}

		if entry, ok := d.xrefIdx[objNum]; ok {
			return entry, nil
		}
	}

	return xrefEntry{}, fmt.Errorf("object %d not found in xref", objNum)
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

	xrefStartOffset, _ := readStartxref(f)

	return &document{
		ref:             ref,
		file:            f,
		sizeBytes:       st.Size(),
		declaredVersion: version,
		xrefStartOffset: xrefStartOffset,
	}, nil
}

func (s *service) Info(_ context.Context, d doc.Document) (doc.InfoResult, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return doc.InfoResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	info := doc.InfoResult{
		Format:          pdfDoc.ref.Format,
		Path:            pdfDoc.ref.Path,
		SizeBytes:       pdfDoc.sizeBytes,
		DeclaredVersion: pdfDoc.declaredVersion,
	}

	if pdfDoc.file != nil {
		if xrefOffset, err := readStartxref(pdfDoc.file); err == nil {
			if ids, err := readTrailerID(pdfDoc.file, xrefOffset); err == nil {
				info.FileIdentifiers = ids
			}
			title, author, creator, producer := readInfoMetadata(pdfDoc.file, xrefOffset)
			info.Title = title
			info.Author = author
			info.Creator = creator
			info.Producer = producer
		}
		if count, err := ReadPageCount(pdfDoc.file); err == nil {
			info.PageCount = count
		}
	}

	return info, nil
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

	if err := ValidateDeep(pdfDoc.file); err != nil {
		report.Valid = false
		report.Errors = append(report.Errors, err.Error())
	}

	return report, nil
}

// PageInfo holds raw PDF object references and attributes for a single page.
type PageInfo struct {
	PageRef  PDFRef
	Parent   PDFRef
	MediaBox PDFArray
	Resources PDFRef
	InlineResources PDFDict
	Contents []PDFRef
	Rotate   *int64
}

// readAllPages traverses the PDF Pages tree and collects info for every page.
func readAllPages(f *os.File, pagesRef string) ([]PageInfo, error) {
	return readAllPagesDepth(f, pagesRef, 0, map[string]struct{}{})
}

func readAllPagesDepth(f *os.File, pagesRef string, depth int, visited map[string]struct{}) ([]PageInfo, error) {
	if depth > maxPageTreeDepth {
		return nil, fmt.Errorf("readAllPages: pages tree depth exceeds maximum %d", maxPageTreeDepth)
	}

	pagesObj, err := readObject(f, pagesRef)
	if err != nil {
		return nil, fmt.Errorf("readAllPages: %w", err)
	}

	pagesDict, err := extractDictFromObject(pagesObj)
	if err != nil {
		return nil, fmt.Errorf("readAllPages: %w", err)
	}

	typ, ok := DictGetName(pagesDict, "Type")
	if !ok {
		return nil, fmt.Errorf("readAllPages: /Type key not found in pages object %s", pagesRef)
	}
	if typ != "Pages" {
		return nil, fmt.Errorf("readAllPages: object %s is /Type /%s, expected /Type /Pages", pagesRef, typ)
	}

	kidsArr, ok := DictGetArray(pagesDict, "Kids")
	if !ok {
		return nil, fmt.Errorf("readAllPages: /Kids not found in pages object %s", pagesRef)
	}

	refs := ArrayToRefs(kidsArr)
	if len(refs) == 0 {
		return nil, fmt.Errorf("readAllPages: pages object %s has empty /Kids", pagesRef)
	}

	var result []PageInfo
	for _, ref := range refs {
		refStr := RefToString(ref)
		if _, seen := visited[refStr]; seen {
			continue
		}
		visited[refStr] = struct{}{}

		kidObj, err := readObject(f, refStr)
		if err != nil {
			continue
		}

		kidDict, err := extractDictFromObject(kidObj)
		if err != nil {
			continue
		}

		kidType, ok := DictGetName(kidDict, "Type")
		if !ok {
			continue
		}

		if kidType == "Pages" {
			pages, err := readAllPagesDepth(f, refStr, depth+1, visited)
			if err == nil {
				result = append(result, pages...)
			}
			continue
		}

		if kidType == "Page" {
			parent, ok := DictGetRef(kidDict, "Parent")
			if !ok {
				continue
			}

			mediaBox, ok := DictGetArray(kidDict, "MediaBox")
			if !ok {
				mediaBox, _ = lookupMediaBoxFromAncestors(refStr, f)
			}

			resources, ok := DictGetRef(kidDict, "Resources")
			inlineResources, hasInline := DictGetDict(kidDict, "Resources")
			if !ok && !hasInline {
				resources, _ = lookupResourcesFromAncestors(refStr, f)
			}

			contents, err := readPageContentsRefs(kidDict)
			if err != nil || len(contents) == 0 {
				continue
			}

			rotate, _ := lookupRotateFromAncestors(refStr, f)

			result = append(result, PageInfo{
				PageRef:         ref,
				Parent:          parent,
				MediaBox:        mediaBox,
				Resources:       resources,
				InlineResources: inlineResources,
				Contents:        contents,
				Rotate:          rotate,
			})
		}
	}

	return result, nil
}

func (s *service) ExtractText(_ context.Context, d doc.Document) (doc.TextResult, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return doc.TextResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	xrefOffset, err := readStartxref(pdfDoc.file)
	if err != nil {
		return doc.TextResult{}, fmt.Errorf("text extraction: %w", err)
	}

	rootRefStr, err := readTrailerRootRef(pdfDoc.file, xrefOffset)
	if err != nil {
		return doc.TextResult{}, fmt.Errorf("text extraction: %w", err)
	}

	catalogObj, err := readObject(pdfDoc.file, rootRefStr)
	if err != nil {
		return doc.TextResult{}, fmt.Errorf("text extraction: %w", err)
	}

	pagesRefStr, err := readPagesRefFromCatalog(catalogObj)
	if err != nil {
		return doc.TextResult{}, fmt.Errorf("text extraction: %w", err)
	}

	pages, err := readAllPages(pdfDoc.file, pagesRefStr)
	if err != nil {
		return doc.TextResult{}, fmt.Errorf("text extraction: %w", err)
	}

	var text strings.Builder
	var lastErr error
	for _, page := range pages {
		for _, contentRef := range page.Contents {
			ref := PDFRef{ObjNum: contentRef.ObjNum, GenNum: contentRef.GenNum}
			streamData, err := readContentStream(pdfDoc.file, ref)
			if err != nil {
				lastErr = err
				continue
			}
			extracted := extractLiteralStrings(streamData)
			if extracted != "" {
				text.WriteString(extracted)
				text.WriteString(" ")
			}
		}
	}

	if text.Len() == 0 {
		if lastErr != nil {
			return doc.TextResult{}, fmt.Errorf("text extraction: %v", lastErr)
		}
		return doc.TextResult{}, fmt.Errorf("text extraction is not implemented for PDF")
	}
	return doc.TextResult{Text: strings.TrimSpace(text.String())}, nil
}

func readContentStream(f *os.File, ref PDFRef) ([]byte, error) {
	xrefOffset, err := readStartxref(f)
	if err != nil {
		return nil, fmt.Errorf("readStartxref: %w", err)
	}

	objOffset, err := findObjectOffsetInXref(f, xrefOffset, strconv.FormatInt(ref.ObjNum, 10))
	if err != nil {
		return nil, fmt.Errorf("find object %d offset: %w", ref.ObjNum, err)
	}

	_, err = f.Seek(objOffset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	rd := bufio.NewReader(f)

	headerLine, err := rd.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}
	headerLine = strings.TrimRight(headerLine, "\r\n")

	expectedPrefix := fmt.Sprintf("%d %d obj", ref.ObjNum, ref.GenNum)
	if !strings.HasPrefix(headerLine, expectedPrefix) {
		return nil, fmt.Errorf("expected object %s, got %q", expectedPrefix, headerLine)
	}

	var dictAndStream strings.Builder
	dictAndStream.WriteString(headerLine)

	openBrackets := strings.Count(headerLine, "<<") - strings.Count(headerLine, ">>")
	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		dictAndStream.WriteString(line)
		if line == "endobj" || strings.HasPrefix(line, "endobj") {
			break
		}
		openBrackets += strings.Count(line, "<<") - strings.Count(line, ">>")
		if openBrackets == 0 && strings.Contains(dictAndStream.String(), ">>stream") {
			break
		}
	}

	dictStr := dictAndStream.String()

	streamIdx := strings.Index(dictStr, ">>stream")
	if streamIdx < 0 {
		return nil, fmt.Errorf("no stream found in object")
	}

	streamDataStart := streamIdx + len(">>stream")
	if streamDataStart < len(dictStr) && dictStr[streamDataStart] == '\r' {
		streamDataStart++
	}
	if streamDataStart < len(dictStr) && dictStr[streamDataStart] == '\n' {
		streamDataStart++
	}

	lengthIdx := strings.Index(dictStr, "/Length ")
	length := 0
	if lengthIdx >= 0 && lengthIdx < streamIdx {
		lengthPart := dictStr[lengthIdx+8:]
		lengthEnd := 0
		for lengthEnd < len(lengthPart) && lengthPart[lengthEnd] >= '0' && lengthPart[lengthEnd] <= '9' {
			lengthEnd++
		}
		if lengthEnd > 0 {
			fmt.Sscanf(lengthPart[:lengthEnd], "%d", &length)
		}
		rest := strings.TrimSpace(lengthPart[lengthEnd:])
		if strings.HasPrefix(rest, "0 R") {
			lengthObjNum := int64(length)
			lengthRefStr := fmt.Sprintf("%d 0 R", lengthObjNum)
			lengthObjStr, err := readObject(f, lengthRefStr)
			if err == nil {
				lengthDict, err := extractDictFromObject(lengthObjStr)
				if err == nil {
					lengthFromDict, ok := DictGetInt(lengthDict, "Length")
					if ok {
						length = int(lengthFromDict)
					}
				} else {
					lengthObjStr = strings.TrimSpace(lengthObjStr)
					lines := strings.Split(lengthObjStr, "\n")
					for _, line := range lines {
						line = strings.TrimSpace(line)
						if len(line) == 0 || strings.Contains(line, "obj") || strings.Contains(line, "endobj") {
							continue
						}
						if _, err := fmt.Sscanf(line, "%d", &length); err == nil {
							break
						}
					}
				}
			}
		}
	}

	streamDataFilePos := objOffset + int64(len(dictStr)) + 3
	_, err = f.Seek(streamDataFilePos, io.SeekStart)
	if err != nil {
		return nil, err
	}
	rd = bufio.NewReader(f)

	streamBuf := make([]byte, 0, 4096)
	if length > 0 {
		// Guard against malicious or corrupt /Length values that would
		// cause an unbounded allocation and potential OOM.
		if length > maxStreamLength {
			return nil, fmt.Errorf("stream length %d exceeds maximum %d bytes", length, maxStreamLength)
		}
		streamBuf = make([]byte, length)
		n, err := io.ReadFull(rd, streamBuf)
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, fmt.Errorf("read stream data: %w", err)
		}
		streamBuf = streamBuf[:n]
	} else {
		data, err := io.ReadAll(rd)
		if err != nil {
			return nil, fmt.Errorf("read stream data: %w", err)
		}
		streamBuf = data
	}

	if strings.Contains(dictStr, "/Filter /FlateDecode") || strings.Contains(dictStr, "/Filter/FlateDecode") {
		r, err := zlib.NewReader(bytes.NewReader(streamBuf))
		if err != nil {
			return nil, fmt.Errorf("zlib NewReader: %w", err)
		}
		decompressed, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("zlib ReadAll: %w", err)
		}
		return decompressed, nil
	}

	return streamBuf, nil
}

// extractLiteralStrings extracts and decodes literal and hex strings from raw PDF stream content.
// It scans for literal strings (parentheses) and hex strings (angle brackets), decoding UTF-16 BOM
// and escape sequences where present. Results are joined with spaces.
func extractLiteralStrings(data []byte) string {
	var result strings.Builder
	s := string(data)

	for {
		literalStart := strings.Index(s, "(")
		hexStart := strings.Index(s, "<")
		if literalStart < 0 && hexStart < 0 {
			break
		}

		if hexStart >= 0 && (literalStart < 0 || hexStart < literalStart) {
			end := hexStart + 1
			for end < len(s) && s[end] != '>' {
				end++
			}
			if end < len(s) && s[end] == '>' {
				hex := s[hexStart+1 : end]
				if isHexString(hex) {
					decoded := decodeHexString(hex)
					if decoded != "" {
						if result.Len() > 0 {
							result.WriteString(" ")
						}
						result.WriteString(decoded)
					}
				}
			}
			s = s[hexStart+1:]
			continue
		}

		if literalStart >= 0 {
			end := literalStart + 1
			parenDepth := 1
			for end < len(s) && parenDepth > 0 {
				if s[end] == '\\' && end+1 < len(s) {
					end += 2
					continue
				}
				if s[end] == '(' {
					parenDepth++
				} else if s[end] == ')' {
					parenDepth--
				}
				end++
			}
			if parenDepth == 0 {
				literal := s[literalStart:end]
				literal = strings.Trim(literal, "()")
				if len(literal) > 0 {
					unescaped := unescapeLiteralString(literal)
					b := []byte(unescaped)
					var decoded string
					if len(b) >= 2 {
						if b[0] == 0xFE && b[1] == 0xFF {
							decoded = utf16BEToUTF8(b[2:])
						} else if b[0] == 0xFF && b[1] == 0xFE {
							decoded = utf16LEToUTF8(b[2:])
						}
					}
					if decoded == "" {
						decoded = unescaped
					}
					if result.Len() > 0 {
						result.WriteString(" ")
					}
					result.WriteString(decoded)
				}
			}
			s = s[literalStart+1:]
			continue
		}

		break
	}

	return result.String()
}

func isHexString(s string) bool {
	for _, c := range s {
		if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return len(s) > 0
}

func decodeHexStringRaw(s string) []byte {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\t", "")
	if len(s)%2 != 0 {
		s = s + "0"
	}
	result := make([]byte, 0, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		h := s[i : i+2]
		b, err := strconv.ParseUint(h, 16, 8)
		if err != nil {
			continue
		}
		result = append(result, byte(b))
	}
	return result
}

// decodeHexString decodes a PDF hex string (angle-bracket notation) into a UTF-8 string.
// It detects and handles UTF-16BE BOM (0xFE 0xFF), UTF-16LE BOM (0xFF 0xFE), and uses a heuristic
// to detect no-BOM UTF-16BE (even-length data with alternating 0x00 bytes). Falls back to
// printable ASCII filtering if no Unicode encoding is detected.
func decodeHexString(s string) string {
	decoded := decodeHexStringRaw(s)
	if len(decoded) == 0 {
		return ""
	}
	if decoded[0] == 0xFE && decoded[1] == 0xFF {
		return utf16BEToUTF8(decoded[2:])
	}
	if decoded[0] == 0xFF && decoded[1] == 0xFE {
		return utf16LEToUTF8(decoded[2:])
	}
	// Heuristic: no-BOM UTF-16BE has alternating 0x00 bytes (odd indices).
	// If more than 25% of odd-indexed bytes are 0x00, assume UTF-16BE and decode directly.
	if len(decoded) >= 4 && len(decoded)%2 == 0 {
		zeroCount := 0
		for i := 1; i < len(decoded); i += 2 {
			if decoded[i] == 0x00 {
				zeroCount++
			}
		}
		if zeroCount > len(decoded)/4 {
			return utf16BEToUTF8(decoded)
		}
	}
	if isMostlyPrintableASCII(decoded) {
		return printableASCIIOnly(decoded)
	}
	return ""
}

func isMostlyPrintableASCII(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	printable := 0
	for _, b := range data {
		if b >= 0x20 && b <= 0x7E {
			printable++
		}
	}
	return printable > len(data)/2
}

func printableASCIIOnly(data []byte) string {
	var result strings.Builder
	for _, b := range data {
		if b >= 0x20 && b <= 0x7E {
			result.WriteByte(b)
		} else if b == '\n' || b == '\r' || b == '\t' {
			result.WriteByte(b)
		} else {
			result.WriteByte(' ')
		}
	}
	return strings.TrimSpace(result.String())
}

// utf16BEToUTF8 converts a UTF-16 BE byte slice to a UTF-8 string.
func utf16BEToUTF8(data []byte) string {
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data)-1; i += 2 {
		runes = append(runes, rune(data[i])<<8|rune(data[i+1]))
	}
	return string(runes)
}

// utf16LEToUTF8 converts a UTF-16 LE byte slice to a UTF-8 string.
func utf16LEToUTF8(data []byte) string {
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i+1 < len(data); i += 2 {
		runes = append(runes, rune(data[i])|rune(data[i+1])<<8)
	}
	return string(runes)
}

// unescapeLiteralString processes PDF literal string escape sequences.
// Handles: \n \r \t \b \f \( \) \\ and \ddd (up to 3 octal digits).
func unescapeLiteralString(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var buf strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '\\' || i+1 >= len(s) {
			buf.WriteByte(s[i])
			i++
			continue
		}
		i++
		switch s[i] {
		case 'n':
			buf.WriteByte('\n')
			i++
		case 'r':
			buf.WriteByte('\r')
			i++
		case 't':
			buf.WriteByte('\t')
			i++
		case 'b':
			buf.WriteByte('\b')
			i++
		case 'f':
			buf.WriteByte('\f')
			i++
		case '(':
			buf.WriteByte('(')
			i++
		case ')':
			buf.WriteByte(')')
			i++
		case '\\':
			buf.WriteByte('\\')
			i++
		default:
			if s[i] >= '0' && s[i] <= '7' {
				octal := 0
				for j := 0; j < 3 && i < len(s) && s[i] >= '0' && s[i] <= '7'; j++ {
					octal = octal*8 + int(s[i]-'0')
					i++
				}
				buf.WriteByte(byte(octal))
			} else {
				buf.WriteByte(s[i])
				i++
			}
		}
	}
	return buf.String()
}

func (s *service) RenderPreview(_ context.Context, d doc.Document, _ doc.PreviewRequest) (doc.PreviewResult, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return doc.PreviewResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	_ = pdfDoc
	return doc.PreviewResult{}, fmt.Errorf("preview is not implemented for %q", doc.FormatPDF)
}

func (s *service) FirstPageInfo(_ context.Context, d doc.Document) (*doc.FirstPageInfoResult, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return nil, fmt.Errorf("unsupported document type %T", d)
	}

	if pdfDoc.file == nil {
		return nil, fmt.Errorf("document file is not open")
	}

	_, err := pdfDoc.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	info, err := ReadFirstPageInfo(pdfDoc.file)
	if err != nil {
		return nil, err
	}

	return toFirstPageInfoResult(info), nil
}

// NewPageIterator implements doc.PageIteratorProvider.
func (s *service) NewPageIterator(ctx context.Context, d doc.Document) (doc.PageIterator, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return nil, fmt.Errorf("NewPageIterator: unsupported document type %T", d)
	}
	iter, err := newPageIterator(pdfDoc)
	if err != nil {
		return nil, err
	}
	return &pdfPageIterator{iter: iter}, nil
}

// NewNavigator implements doc.NavigatorProvider.
func (s *service) NewNavigator(ctx context.Context, d doc.Document) (doc.Navigator, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return nil, fmt.Errorf("NewNavigator: unsupported document type %T", d)
	}
	return newPDFNavigator(pdfDoc), nil
}

func toFirstPageInfoResult(info *FirstPageInfo) *doc.FirstPageInfoResult {
	contents := make([]doc.RefInfo, len(info.Contents))
	for i, c := range info.Contents {
		contents[i] = doc.RefInfo{ObjNum: c.ObjNum, GenNum: c.GenNum}
	}

	mediaBox := make([]float64, len(info.MediaBox))
	for i, v := range info.MediaBox {
		if f, ok := v.(PDFReal); ok {
			mediaBox[i] = float64(f)
		} else if iv, ok := v.(PDFInteger); ok {
			mediaBox[i] = float64(iv)
		}
	}

	return &doc.FirstPageInfoResult{
		Path:     "",
		PagesRef: doc.RefInfo{ObjNum: info.PagesRef.ObjNum, GenNum: info.PagesRef.GenNum},
		PageRef:  doc.RefInfo{ObjNum: info.PageRef.ObjNum, GenNum: info.PageRef.GenNum},
		Parent:   doc.RefInfo{ObjNum: info.Parent.ObjNum, GenNum: info.Parent.GenNum},
		MediaBox: mediaBox,
		Resources: doc.RefInfo{
			ObjNum: info.Resources.ObjNum,
			GenNum: info.Resources.GenNum,
		},
		Contents: contents,
		Rotate:   info.Rotate,
	}
}

func (s *service) Save(_ context.Context, ref doc.DocumentRef, dst string) error {
	return CopyFile(ref.Path, dst)
}

func (s *service) Sign(_ context.Context, d doc.Document, _ doc.SignRequest) (doc.SignResult, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return doc.SignResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	_ = pdfDoc
	return doc.SignResult{}, fmt.Errorf("signing is not implemented for %q", doc.FormatPDF)
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

func readStartxref(f *os.File) (int64, error) {
	info, err := f.Stat()
	if err != nil {
		return 0, err
	}

	size := info.Size()
	if size < 20 {
		return 0, fmt.Errorf("file too small for PDF startxref")
	}

	searchLen := 1024
	if int64(searchLen) > size {
		searchLen = int(size)
	}

	buf := make([]byte, searchLen)
	_, err = f.Seek(size-int64(searchLen), io.SeekStart)
	if err != nil {
		return 0, err
	}

	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}

	searchArea := string(buf[:n])
	idx := strings.LastIndex(searchArea, "startxref")
	if idx < 0 {
		return 0, fmt.Errorf("startxref not found")
	}

	afterStartxref := searchArea[idx+len("startxref"):]
	lines := strings.Split(afterStartxref, "\n")
	var offsetStr string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			offsetStr = trimmed
			break
		}
	}
	if offsetStr == "" {
		return 0, fmt.Errorf("startxref: offset not found")
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("startxref: invalid offset %q: %w", offsetStr, err)
	}

	return offset, nil
}

func readTrailerRootRef(f *os.File, xrefOffset int64) (string, error) {
	_, err := f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("seek to xref at %d: %w", xrefOffset, err)
	}

	rd := bufio.NewReader(f)

	trailerDict, isXRefStream, err := readTrailerDictLines(rd)
	if err != nil {
		return "", fmt.Errorf("read trailer dict: %w", err)
	}

	if isXRefStream {
		d, err := ParseDictContent(trailerDict)
		if err != nil {
			return "", fmt.Errorf("parse xref stream dict: %w", err)
		}

		ref, ok := DictGetRef(d, "Root")
		if ok {
			return RefToString(ref), nil
		}

		prevOffset, hasPrev := DictGetInt(d, "Prev")
		if hasPrev {
			return readTrailerRootRef(f, prevOffset)
		}

		return "", fmt.Errorf("Root key not found in xref stream")
	}

	if trailerDict == "" {
		return "", fmt.Errorf("trailer not found after xref at %d", xrefOffset)
	}

	d, err := ParseDictContent(trailerDict)
	if err != nil {
		return "", fmt.Errorf("parse trailer dict: %w", err)
	}

	ref, ok := DictGetRef(d, "Root")
	if ok {
		return RefToString(ref), nil
	}

	prevOffset, hasPrev := DictGetInt(d, "Prev")
	if hasPrev {
		return readTrailerRootRef(f, prevOffset)
	}

	return "", fmt.Errorf("Root key not found in trailer")
}

func readTrailerID(f *os.File, xrefOffset int64) ([]string, error) {
	_, err := f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seek to xref at %d: %w", xrefOffset, err)
	}

	rd := bufio.NewReader(f)

	trailerDict, isXRefStream, err := readTrailerDictLines(rd)
	if err != nil {
		return nil, fmt.Errorf("read trailer dict: %w", err)
	}

	if isXRefStream {
		d, err := ParseDictContent(trailerDict)
		if err != nil {
			return nil, fmt.Errorf("parse xref stream dict: %w", err)
		}

		if arr, ok := DictGetArray(d, "ID"); ok {
			return arrayToStrings(arr), nil
		}
		return nil, nil
	}

	if trailerDict == "" {
		return nil, nil
	}

	d, err := ParseDictContent(trailerDict)
	if err != nil {
		return nil, fmt.Errorf("parse trailer dict: %w", err)
	}

	if arr, ok := DictGetArray(d, "ID"); ok {
		return arrayToStrings(arr), nil
	}

	return nil, nil
}

func arrayToStrings(arr PDFArray) []string {
	var result []string
	for _, obj := range arr {
		switch v := obj.(type) {
		case PDFLiteralString:
			result = append(result, string(v))
		case PDFHexString:
			result = append(result, string(v))
		case string:
			result = append(result, v)
		}
	}
	return result
}

// decodePDFString decodes a raw PDF string value (after stripping outer delimiters).
// It handles UTF-16BE (BOM 0xFE 0xFF), UTF-16LE (BOM 0xFF 0xFE), literal string escape
// sequences, and falls back to printable ASCII filtering.
func decodePDFString(raw string) string {
	unescaped := unescapeLiteralString(raw)
	b := []byte(unescaped)
	if len(b) >= 2 {
		if b[0] == 0xFE && b[1] == 0xFF {
			return utf16BEToUTF8(b[2:])
		}
		if b[0] == 0xFF && b[1] == 0xFE {
			return utf16LEToUTF8(b[2:])
		}
	}
	if isMostlyPrintableASCII(b) {
		return printableASCIIOnly(b)
	}
	return ""
}

// derefStringValue resolves an indirect object reference to its string content.
// If the object is a PDFRef, it reads the referenced object and extracts the
// PDF string or name literal from it. If the object is already a direct
// PDFLiteralString or PDFHexString, it is decoded and returned unchanged.
func derefStringValue(f *os.File, obj PDFObject) string {
	ref, ok := obj.(PDFRef)
	if !ok {
		if ls, ok := obj.(PDFLiteralString); ok {
			return decodePDFString(string(ls))
		}
		if hs, ok := obj.(PDFHexString); ok {
			return decodeHexString(string(hs))
		}
		return ""
	}

	refStr := RefToString(ref)
	objStr, err := readObject(f, refStr)
	if err != nil {
		return ""
	}

	return extractStringFromObjContent(objStr)
}

// extractStringFromObjContent extracts a PDF string or name from raw object content.
// The object content format is "N G obj\n<content>\nendobj".
// It handles literal strings (...), hex strings <...>, and name literals /Name.
func extractStringFromObjContent(objStr string) string {
	lines := strings.Split(objStr, "\n")
	if len(lines) < 2 {
		return ""
	}
	content := strings.TrimSpace(lines[1])
	if len(content) == 0 {
		return ""
	}

	if len(content) >= 2 && content[0] == '(' && content[len(content)-1] == ')' {
		return decodePDFString(content[1 : len(content)-1])
	}
	if len(content) >= 2 && content[0] == '<' && content[len(content)-1] == '>' {
		return decodeHexString(content[1 : len(content)-1])
	}
	if len(content) >= 1 && content[0] == '/' {
		return content[1:]
	}
	return ""
}

func readInfoMetadata(f *os.File, xrefOffset int64) (title, author, creator, producer string) {
	infoRef, err := readTrailerInfoRef(f, xrefOffset)
	if err != nil || infoRef == "" {
		return "", "", "", ""
	}

	infoObj, err := readObject(f, infoRef)
	if err != nil {
		return "", "", "", ""
	}

	infoDict, err := extractDictFromObject(infoObj)
	if err != nil {
		return "", "", "", ""
	}

	if obj := DictGet(infoDict, "Title"); obj != nil {
		title = derefStringValue(f, obj)
	}

	if obj := DictGet(infoDict, "Author"); obj != nil {
		author = derefStringValue(f, obj)
	}

	if obj := DictGet(infoDict, "Creator"); obj != nil {
		creator = derefStringValue(f, obj)
	}

	if obj := DictGet(infoDict, "Producer"); obj != nil {
		producer = derefStringValue(f, obj)
	}

	return title, author, creator, producer
}

func readTrailerInfoRef(f *os.File, xrefOffset int64) (string, error) {
	_, err := f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("seek to xref at %d: %w", xrefOffset, err)
	}

	rd := bufio.NewReader(f)

	trailerDict, isXRefStream, err := readTrailerDictLines(rd)
	if err != nil {
		return "", fmt.Errorf("read trailer dict: %w", err)
	}

	if isXRefStream {
		d, err := ParseDictContent(trailerDict)
		if err != nil {
			return "", fmt.Errorf("parse xref stream dict: %w", err)
		}

		if ref, ok := DictGetRef(d, "Info"); ok {
			return RefToString(ref), nil
		}
		return "", nil
	}

	if trailerDict == "" {
		return "", nil
	}

	d, err := ParseDictContent(trailerDict)
	if err != nil {
		return "", fmt.Errorf("parse trailer dict: %w", err)
	}

	if ref, ok := DictGetRef(d, "Info"); ok {
		return RefToString(ref), nil
	}

	return "", nil
}

func readTrailerDictLines(rd *bufio.Reader) (string, bool, error) {
	isXRefStream := false
	afterObjHeader := false
	combined := ""

	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false, err
		}
		line = strings.TrimRight(line, "\r\n")

		if strings.TrimSpace(line) == "" {
			continue
		}

		if line == "trailer" || strings.HasPrefix(line, "trailer") {
			var trailerDict string
			if strings.HasPrefix(line, "trailer<<") {
				trailerDict = line[len("trailer"):]
			}

			dictLine, err := rd.ReadString('\n')
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", false, err
			}
			dictLine = strings.TrimRight(dictLine, "\r\n")
			if trailerDict == "" {
				trailerDict = dictLine
			} else {
				trailerDict += dictLine
			}

			openBrackets := strings.Count(trailerDict, "<<") - strings.Count(trailerDict, ">>")
			for openBrackets > 0 {
				nextLine, err := rd.ReadString('\n')
				if err == io.EOF {
					break
				}
				if err != nil {
					return "", false, err
				}
				nextLine = strings.TrimRight(nextLine, "\r\n")
				trailerDict += nextLine
				openBrackets += strings.Count(nextLine, "<<") - strings.Count(nextLine, ">>")
			}

			return trailerDict, false, nil
		}

		combined += line

		if !afterObjHeader {
			idx := strings.Index(line, "<<")
			if idx >= 0 {
				afterObjHeader = true
				combined = line[idx:]
				continue
			}
			idxObj := strings.Index(line, " obj")
			if idxObj >= 0 {
				afterObjHeader = true
				rest := line[idxObj+4:]
				idx = strings.Index(rest, "<<")
				if idx >= 0 {
					combined = rest[idx:]
				}
				continue
			}
		}

		if strings.Contains(line, "Type") && strings.Contains(line, "XRef") {
			isXRefStream = true
		}
	}

	if combined == "" {
		return "", false, nil
	}

	firstDictIdx := strings.Index(combined, "<<")
	if firstDictIdx < 0 {
		return "", false, nil
	}

	dictContent := combined[firstDictIdx:]

	idxStream := strings.Index(dictContent, ">>stream")
	if idxStream >= 0 {
		dictContent = dictContent[:idxStream+2]
		openBrackets := strings.Count(dictContent, "<<") - strings.Count(dictContent, ">>")
		if openBrackets == 0 {
			return dictContent, isXRefStream, nil
		}
	}

	openBrackets := strings.Count(dictContent, "<<") - strings.Count(dictContent, ">>")
	for openBrackets > 0 {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", false, err
		}
		line = strings.TrimRight(line, "\r\n")
		dictContent += line
		openBrackets += strings.Count(line, "<<") - strings.Count(line, ">>")
	}

	return dictContent, isXRefStream, nil
}

func skipXrefTable(rd *bufio.Reader) error {
	for {
		line, err := rd.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		line = strings.TrimRight(line, "\r\n")

		if strings.TrimSpace(line) == "trailer" {
			pushBackLine(rd, line)
			return nil
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			startObj, _ := strconv.ParseInt(parts[0], 10, 64)
			count, _ := strconv.ParseInt(parts[1], 10, 64)
			if startObj == 0 && count > 0 {
				for i := int64(0); i < count; i++ {
					entryLine, err := rd.ReadString('\n')
					if err != nil && err != io.EOF {
						return err
					}
					_ = entryLine
				}
				continue
			}
			for i := int64(0); i < count; i++ {
				entryLine, err := rd.ReadString('\n')
				if err != nil && err != io.EOF {
					return err
				}
				_ = entryLine
			}
			continue
		}

		if strings.TrimSpace(line) == "" {
			continue
		}
	}
}

func pushBackLine(rd *bufio.Reader, line string) {
	rd.UnreadByte()
	for i := len(line) - 1; i >= 0; i-- {
		rd.UnreadByte()
	}
}

func readTrailerDict(rd *bufio.Reader) (string, error) {
	var dict strings.Builder
	openBrackets := 0

	for {
		line, err := rd.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")

		for _, ch := range line {
			if ch == '<' {
				openBrackets++
			} else if ch == '>' {
				openBrackets--
			}
		}

		dict.WriteString(line)

		if openBrackets <= 0 {
			break
		}
	}

	return dict.String(), nil
}

// readObject reads a PDF object using the unified xref index.
// For backward compatibility, if the object is not found in the index,
// it falls back to the legacy findObjectOffsetInXref method.
func readObject(f *os.File, ref string) (string, error) {
	parts := strings.Fields(ref)
	if len(parts) < 3 || parts[2] != "R" {
		return "", fmt.Errorf("invalid object ref %q", ref)
	}
	objNumStr := parts[0]
	genNum := parts[1]
	objNum, _ := strconv.ParseInt(objNumStr, 10, 64)

	// Build xref index for this file (cached if called multiple times)
	xrefOffset, err := readStartxref(f)
	if err != nil {
		return "", fmt.Errorf("readStartxref for object lookup: %w", err)
	}
	idx, err := buildXRefIndex(f, xrefOffset)
	if err != nil {
		// Fallback to legacy method if index build fails
		return readObjectLegacy(f, ref, xrefOffset)
	}

	// Resolve object from index
	objData, err := resolveObject(f, idx, objNum)
	if err != nil {
		// Fallback to legacy method
		return readObjectLegacy(f, ref, xrefOffset)
	}

	// Validate that the object header matches expected
	objStr := string(objData)
	lines := strings.Split(objStr, "\n")
	if len(lines) > 0 {
		expectedPrefix := objNumStr + " " + genNum + " obj"
		headerLineTrimmed := strings.TrimRight(lines[0], "\r\n")
		if !strings.HasPrefix(headerLineTrimmed, expectedPrefix) {
			// The resolved object might not have the header (e.g., from ObjStm)
			// Wrap it with the proper header
			return expectedPrefix + "\n<<" + objStr + ">>\nendobj\n", nil
		}
	}

	return objStr, nil
}

// readObjectLegacy uses the legacy xref lookup method for backward compatibility.
func readObjectLegacy(f *os.File, ref string, xrefOffset int64) (string, error) {
	parts := strings.Fields(ref)
	objNum := parts[0]
	genNum := parts[1]

	objOffset, err := findObjectOffsetInXref(f, xrefOffset, objNum)
	if err != nil {
		return "", fmt.Errorf("find object %s offset: %w", objNum, err)
	}

	_, err = f.Seek(objOffset, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("seek to object at %d: %w", objOffset, err)
	}

	objRd := bufio.NewReader(f)
	headerLine, err := objRd.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	expectedPrefix := objNum + " " + genNum + " obj"
	headerLineTrimmed := strings.TrimRight(headerLine, "\r\n")
	if !strings.HasPrefix(headerLineTrimmed, expectedPrefix) {
		return "", fmt.Errorf("expected object %s, got %q", expectedPrefix, headerLineTrimmed)
	}

	var obj strings.Builder
	obj.WriteString(headerLine)

	openBrackets := strings.Count(headerLineTrimmed, "<<") - strings.Count(headerLineTrimmed, ">>")
	for {
		line, err := objRd.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		obj.WriteString(line)
		openBrackets += strings.Count(line, "<<") - strings.Count(line, ">>")
		if openBrackets == 0 {
			break
		}
	}

	return obj.String(), nil
}

func readPagesRefFromCatalog(catalogStr string) (string, error) {
	d, err := extractDictFromObject(catalogStr)
	if err != nil {
		return "", fmt.Errorf("readPagesRefFromCatalog: %w", err)
	}

	if ref, ok := DictGetRef(d, "Pages"); ok {
		return RefToString(ref), nil
	}

	if arr, ok := DictGetArray(d, "Pages"); ok {
		refs := ArrayToRefs(arr)
		if len(refs) > 0 {
			return RefToString(refs[0]), nil
		}
		return "", fmt.Errorf("/Pages array has no reference")
	}

	return "", fmt.Errorf("/Pages key not found in Catalog")
}

func readPagesKids(pagesStr string) ([]string, int, error) {
	dict, err := extractDictFromObject(pagesStr)
	if err != nil {
		return nil, 0, fmt.Errorf("readPagesKids: %w", err)
	}

	kidsArr, ok := DictGetArray(dict, "Kids")
	if !ok {
		return nil, 0, fmt.Errorf("/Kids key not found in Pages")
	}

	refs := ArrayToRefs(kidsArr)

	kidsStr := make([]string, len(refs))
	for i, r := range refs {
		kidsStr[i] = RefToString(r)
	}

	declaredCount := len(kidsStr)
	if cnt, ok := DictGetInt(dict, "Count"); ok {
		declaredCount = int(cnt)
	}

	if declaredCount != len(kidsStr) {
		return nil, 0, fmt.Errorf("/Count mismatch: declared %d but Kids has %d", declaredCount, len(kidsStr))
	}

	return kidsStr, declaredCount, nil
}

func extractDictFromObject(objStr string) (PDFDict, error) {
	openIdx := strings.Index(objStr, "<<")
	closeIdx := strings.LastIndex(objStr, ">>")
	if openIdx < 0 || closeIdx < 0 || closeIdx <= openIdx {
		return nil, fmt.Errorf("no dictionary found in object")
	}
	dictContent := objStr[openIdx:]
	return ParseDictContent(dictContent)
}

func readPageFromKids(f *os.File, kidRef string) (string, error) {
	objStr, err := readObject(f, kidRef)
	if err != nil {
		return "", err
	}

	d, err := extractDictFromObject(objStr)
	if err != nil {
		return "", fmt.Errorf("readPageFromKids: %w", err)
	}

	typ, ok := DictGetName(d, "Type")
	if !ok {
		return "", fmt.Errorf("kid %s: /Type key not found", kidRef)
	}
	if typ != "Page" {
		return "", fmt.Errorf("kid %s is not /Type /Page", kidRef)
	}

	return objStr, nil
}

// readFirstPageFromPages traverses the PDF Pages tree to locate the first
// leaf /Type /Page object. It is a thin wrapper around readFirstPageFromPagesDepth
// that preserves the public signature while enforcing a recursion depth limit.
func readFirstPageFromPages(f *os.File, pagesRef string) (string, error) {
	return readFirstPageFromPagesDepth(f, pagesRef, 0)
}

// readFirstPageFromPagesDepth is the internal recursive implementation of
// readFirstPageFromPages. depth tracks how many levels of the Pages tree have
// already been visited; if it exceeds maxPageTreeDepth the function returns an
// error to prevent stack overflow from maliciously deep trees.
func readFirstPageFromPagesDepth(f *os.File, pagesRef string, depth int) (string, error) {
	if depth > maxPageTreeDepth {
		return "", fmt.Errorf("readFirstPageFromPages: pages tree depth exceeds maximum %d", maxPageTreeDepth)
	}

	pagesObj, err := readObject(f, pagesRef)
	if err != nil {
		return "", fmt.Errorf("readFirstPageFromPages: %w", err)
	}

	d, err := extractDictFromObject(pagesObj)
	if err != nil {
		return "", fmt.Errorf("readFirstPageFromPages: %w", err)
	}

	typ, ok := DictGetName(d, "Type")
	if !ok {
		return "", fmt.Errorf("/Type key not found in pages object %s", pagesRef)
	}

	if typ != "Pages" {
		return "", fmt.Errorf("object %s is /Type /%s, expected /Type /Pages", pagesRef, typ)
	}

	kidsArr, ok := DictGetArray(d, "Kids")
	if !ok {
		return "", fmt.Errorf("/Kids not found in pages object %s", pagesRef)
	}

	refs := ArrayToRefs(kidsArr)
	if len(refs) == 0 {
		return "", fmt.Errorf("pages object %s has empty /Kids", pagesRef)
	}

	for _, ref := range refs {
		refStr := RefToString(ref)
		kidObj, err := readObject(f, refStr)
		if err != nil {
			return "", fmt.Errorf("readFirstPageFromPages: %w", err)
		}

		kidDict, err := extractDictFromObject(kidObj)
		if err != nil {
			return "", fmt.Errorf("readFirstPageFromPages: %w", err)
		}

		kidType, ok := DictGetName(kidDict, "Type")
		if !ok {
			continue
		}

		if kidType == "Pages" {
			page, err := readFirstPageFromPagesDepth(f, refStr, depth+1)
			if err == nil {
				return page, nil
			}
			continue
		}

		if kidType == "Page" {
			return kidObj, nil
		}
	}

	return "", fmt.Errorf("no /Type /Page found in pages tree under %s", pagesRef)
}

func readPageResourcesRef(pageDict PDFDict) (PDFRef, error) {
	ref, ok := DictGetRef(pageDict, "Resources")
	if !ok {
		return PDFRef{}, fmt.Errorf("/Resources key not found or not a single indirect ref")
	}
	return ref, nil
}

func readPageContentsRefs(pageDict PDFDict) ([]PDFRef, error) {
	if ref, ok := DictGetRef(pageDict, "Contents"); ok {
		return []PDFRef{ref}, nil
	}
	if arr, ok := DictGetArray(pageDict, "Contents"); ok {
		refs := ArrayToRefs(arr)
		if len(refs) == 0 {
			return nil, fmt.Errorf("/Contents array has no indirect references")
		}
		return refs, nil
	}
	return nil, fmt.Errorf("/Contents key not found and is not a ref or ref array")
}

func findObjectOffsetInXref(f *os.File, xrefOffset int64, objNum string) (int64, error) {
	objNumInt, _ := strconv.ParseInt(objNum, 10, 64)
	return findObjectOffsetInXrefRecursive(f, xrefOffset, objNumInt, map[int64]bool{})
}

func findObjectOffsetInXrefRecursive(f *os.File, xrefOffset int64, objNumInt int64, visited map[int64]bool) (int64, error) {
	if visited[xrefOffset] {
		return 0, fmt.Errorf("object %d not found in xref (cycle detected)", objNumInt)
	}
	visited[xrefOffset] = true

	_, err := f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	rd := bufio.NewReader(f)

	var prevOffset int64
	var foundOffset int64
	foundInThisXref := false

	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		line = strings.TrimRight(line, "\r\n")

		if line == "trailer" {
			continue
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			startObj, _ := strconv.ParseInt(fields[0], 10, 64)
			count, _ := strconv.ParseInt(fields[1], 10, 64)
			if startObj == 0 && count > 0 {
				for i := int64(0); i < count; i++ {
					entryLine, err := rd.ReadString('\n')
					if err != nil && err != io.EOF {
						return 0, err
					}
					entryFields := strings.Fields(entryLine)
					if len(entryFields) >= 1 && i == objNumInt {
						offset, _ := strconv.ParseInt(entryFields[0], 10, 64)
						foundOffset = offset
						foundInThisXref = true
						break
					}
				}
				if foundInThisXref {
					break
				}
				continue
			}
			if objNumInt >= startObj && objNumInt < startObj+count {
				skipCount := objNumInt - startObj
				for i := int64(0); i < skipCount; i++ {
					_, err := rd.ReadString('\n')
					if err != nil && err != io.EOF {
						return 0, err
					}
				}
				entryLine, err := rd.ReadString('\n')
				if err != nil && err != io.EOF {
					return 0, err
				}
				entryFields := strings.Fields(entryLine)
				if len(entryFields) >= 1 {
					offset, _ := strconv.ParseInt(entryFields[0], 10, 64)
					foundOffset = offset
					foundInThisXref = true
					break
				}
			}
			for i := int64(0); i < count; i++ {
				_, err := rd.ReadString('\n')
				if err != nil && err != io.EOF {
					return 0, err
				}
			}
		}
	}

	if foundInThisXref {
		return foundOffset, nil
	}

	// Check for Prev in trailer
	_, err = f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}
	prevOffset, _ = findPrevInTraditionalXref(f, xrefOffset)

	if prevOffset == 0 {
		// Try xref stream
		objOffset, err := findObjectOffsetInXRefStream(f, xrefOffset, objNumInt)
		if err == nil {
			return objOffset, nil
		}

		// Try file body scan
		offset, err := findObjectOffsetInFileBody(f, objNumInt)
		if err == nil {
			return offset, nil
		}

		return 0, fmt.Errorf("object %d not found in xref", objNumInt)
	}

	// Recursively search Prev xref
	return findObjectOffsetInXrefRecursive(f, prevOffset, objNumInt, visited)
}

func findPrevInTraditionalXref(f *os.File, xrefOffset int64) (int64, error) {
	_, err := f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	rd := bufio.NewReader(f)
	var trailerDict string
	var foundTrailer bool

	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		line = strings.TrimRight(line, "\r\n")

		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.HasPrefix(line, "trailer") || line == "trailer" {
			trailerDict = ""
			if strings.HasPrefix(line, "trailer<<") {
				trailerDict = line[len("trailer"):]
			}
			foundTrailer = true
		} else if foundTrailer {
			trailerDict += line

			openBrackets := strings.Count(trailerDict, "<<") - strings.Count(trailerDict, ">>")
			for openBrackets > 0 {
				nextLine, err := rd.ReadString('\n')
				if err == io.EOF {
					break
				}
				if err != nil {
					return 0, err
				}
				nextLine = strings.TrimRight(nextLine, "\r\n")
				trailerDict += nextLine
				openBrackets += strings.Count(nextLine, "<<") - strings.Count(nextLine, ">>")
			}

			idx := strings.Index(trailerDict, "/Prev ")
			if idx >= 0 {
				rest := trailerDict[idx+6:]
				endIdx := 0
				for endIdx < len(rest) && rest[endIdx] >= '0' && rest[endIdx] <= '9' {
					endIdx++
				}
				if endIdx > 0 {
					prev, err := strconv.ParseInt(rest[:endIdx], 10, 64)
					if err == nil {
						return prev, nil
					}
				}
			}
			break
		}
	}

	return 0, nil
}

func findObjectOffsetInFileBody(f *os.File, objNum int64) (int64, error) {
	objStr := fmt.Sprintf("%d 0 obj", objNum)

	_, err := f.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}

	rd := bufio.NewReader(f)
	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		lineTrimmed := strings.TrimRight(line, "\r\n")
		lineTrimmed = strings.TrimRight(lineTrimmed, "\r")

		idx := strings.Index(lineTrimmed, objStr)
		if idx >= 0 {
			before := lineTrimmed[:idx]
			after := ""
			if idx+len(objStr) < len(lineTrimmed) {
				after = lineTrimmed[idx+len(objStr):]
			}
			beforeTrimmed := strings.TrimSpace(before)
			endsWithObjSep := strings.HasSuffix(beforeTrimmed, "endobj") || strings.HasSuffix(beforeTrimmed, "endstream")
			if len(beforeTrimmed) == 0 || endsWithObjSep {
				if len(after) == 0 || strings.HasPrefix(after, " ") || strings.HasPrefix(after, "<") || strings.HasPrefix(after, "[") || strings.HasPrefix(after, "\r") || strings.HasPrefix(after, "\n") {
					filePos, _ := f.Seek(0, io.SeekCurrent)
					return filePos - int64(len(line)) + int64(idx), nil
				}
			}
		}
	}

	return 0, fmt.Errorf("object %d not found in file body scan", objNum)
}

func findObjectOffsetInXRefStream(f *os.File, xrefOffset int64, objNum int64) (int64, error) {
	offset, err := decodeXRefStream(f, xrefOffset, objNum)
	if err == nil {
		return offset, nil
	}

	objStr := fmt.Sprintf("%d 0 obj", objNum)

	_, err = f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	rd := bufio.NewReader(f)
	filePos := xrefOffset

	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		lineTrimmed := strings.TrimRight(line, "\r\n")
		lineTrimmed = strings.TrimRight(lineTrimmed, "\r")

		idx := strings.Index(lineTrimmed, objStr)
		if idx >= 0 {
			before := lineTrimmed[:idx]
			after := ""
			if idx+len(objStr) < len(lineTrimmed) {
				after = lineTrimmed[idx+len(objStr):]
			}
			beforeTrimmed := strings.TrimSpace(before)
			endsWithObjSep := strings.HasSuffix(beforeTrimmed, "endobj") || strings.HasSuffix(beforeTrimmed, "endstream")
			if len(beforeTrimmed) == 0 || endsWithObjSep {
				if len(after) == 0 || strings.HasPrefix(after, " ") || strings.HasPrefix(after, "<") || strings.HasPrefix(after, "[") || strings.HasPrefix(after, "\r") || strings.HasPrefix(after, "\n") {
					return filePos + int64(idx), nil
				}
			}
		}

		filePos += int64(len(line))
	}

	return 0, fmt.Errorf("object %d not found in xref", objNum)
}

func decodeXRefStream(f *os.File, xrefOffset int64, objNum int64) (int64, error) {
	_, err := f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	rd := bufio.NewReader(f)
	xrefData := make([]byte, 0, 500)
	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		xrefData = append(xrefData, line...)
	}

	xrefStr := string(xrefData)

	streamIdx := strings.Index(xrefStr, ">>stream\r\n")
	if streamIdx < 0 {
		return 0, fmt.Errorf("decodeXRefStream: stream marker not found")
	}
	streamDataStart := streamIdx + len(">>stream\r\n")

	wStart := strings.Index(xrefStr, "/W[")
	if wStart < 0 || wStart > streamIdx {
		return 0, fmt.Errorf("decodeXRefStream: /W not found")
	}
	wEnd := strings.Index(xrefStr[wStart:], "]")
	if wEnd < 0 {
		return 0, fmt.Errorf("decodeXRefStream: /W bracket not closed")
	}
	wEnd += wStart + 1
	wStr := xrefStr[wStart+3 : wEnd-1]

	var w [3]int
	if _, err := fmt.Sscanf(wStr, "%d %d %d", &w[0], &w[1], &w[2]); err != nil {
		return 0, fmt.Errorf("decodeXRefStream: parse W: %w", err)
	}
	entrySize := w[0] + w[1] + w[2]
	if entrySize == 0 {
		entrySize = 4
	}

	sizeStart := strings.Index(xrefStr, "/Size ")
	size := 0
	if sizeStart >= 0 && sizeStart < streamIdx {
		sizeEnd := sizeStart + 6
		for sizeEnd < len(xrefStr) && xrefStr[sizeEnd] >= '0' && xrefStr[sizeEnd] <= '9' {
			sizeEnd++
		}
		fmt.Sscanf(xrefStr[sizeStart+6:sizeEnd], "%d", &size)
	}

	firstObj := 0
	count := size
	indexStart := strings.Index(xrefStr, "/Index[")
	if indexStart >= 0 && indexStart < streamIdx {
		indexEnd := strings.Index(xrefStr[indexStart:], "]")
		if indexEnd > 0 {
			indexEnd += indexStart + 1
			fmt.Sscanf(xrefStr[indexStart+7:indexEnd-1], "%d %d", &firstObj, &count)
		}
	}

	lengthStart := strings.Index(xrefStr, "/Length ")
	streamLen := 0
	if lengthStart >= 0 && lengthStart < streamIdx {
		lengthEnd := lengthStart + 8
		for lengthEnd < len(xrefStr) && xrefStr[lengthEnd] >= '0' && xrefStr[lengthEnd] <= '9' {
			lengthEnd++
		}
		fmt.Sscanf(xrefStr[lengthStart+8:lengthEnd], "%d", &streamLen)
	}

	if streamLen <= 0 || streamLen > 10000 {
		return 0, fmt.Errorf("decodeXRefStream: invalid stream length %d", streamLen)
	}

	streamData := xrefData[streamDataStart : streamDataStart+streamLen]

	r, err := zlib.NewReader(bytes.NewReader(streamData))
	if err != nil {
		return 0, fmt.Errorf("decodeXRefStream: zlib NewReader: %w", err)
	}
	decompressed, err := io.ReadAll(r)
	if err != nil {
		return 0, fmt.Errorf("decodeXRefStream: zlib ReadAll: %w", err)
	}

	for i := 0; i < count && i*entrySize+entrySize <= len(decompressed); i++ {
		if int64(firstObj)+int64(i) != objNum {
			continue
		}
		entry := decompressed[i*entrySize : i*entrySize+entrySize]
		typ := int(entry[0])

		if typ == 0 {
			continue
		}

		if typ == 1 && w[1] > 0 {
			var offset uint32
			if w[1] == 1 {
				offset = uint32(entry[1])
			} else if w[1] == 2 {
				offset = uint32(entry[1])<<8 | uint32(entry[2])
			}
			return int64(offset), nil
		}

		if typ == 2 && w[1] > 0 && w[2] > 0 {
			var objStreamNum uint32
			var idxInStream uint32
			pos := 1
			if w[1] == 1 {
				objStreamNum = uint32(entry[pos])
				pos++
			} else if w[1] == 2 {
				objStreamNum = uint32(entry[pos])<<8 | uint32(entry[pos+1])
				pos += 2
			}
			if w[2] == 1 {
				idxInStream = uint32(entry[pos])
			} else if w[2] == 2 {
				idxInStream = uint32(entry[pos])<<8 | uint32(entry[pos+1])
			}
			return decodeCompressedObject(f, int64(objStreamNum), int(idxInStream))
		}
	}

	prevStart := strings.Index(xrefStr, "/Prev ")
	if prevStart >= 0 && prevStart < streamIdx {
		prevEnd := prevStart + 6
		for prevEnd < len(xrefStr) && xrefStr[prevEnd] >= '0' && xrefStr[prevEnd] <= '9' {
			prevEnd++
		}
		var prevOffset int64
		fmt.Sscanf(xrefStr[prevStart+6:prevEnd], "%d", &prevOffset)
		if prevOffset > 0 {
			return decodeXRefStream(f, prevOffset, objNum)
		}
	}

	return 0, fmt.Errorf("decodeXRefStream: object %d not found", objNum)
}

func decodeCompressedObject(f *os.File, objStreamNum int64, idxInStream int) (int64, error) {
	objStr := fmt.Sprintf("%d 0 obj", objStreamNum)

	_, err := f.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}

	rd := bufio.NewReader(f)
	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		line = strings.TrimRight(line, "\r\n")
		line = strings.TrimRight(line, "\r")

		idx := strings.Index(line, objStr)
		if idx >= 0 && idx+len(objStr) <= len(line) {
			after := line[idx+len(objStr):]
			after = strings.TrimSpace(after)
			if len(after) == 0 || strings.HasPrefix(after, "<") || strings.HasPrefix(after, "[") {
				filePos, _ := f.Seek(0, io.SeekCurrent)
				return filePos - int64(len(line)) + int64(idx), nil
			}
		}
	}
	return 0, fmt.Errorf("decodeCompressedObject: stream object %d not found", objStreamNum)
}

func readFirstPageInfoFromPagesObj(f *os.File, pagesObjStr string, pagesRefVal PDFRef) (*FirstPageInfo, error) {
	pagesDict, err := extractDictFromObject(pagesObjStr)
	if err != nil {
		return nil, fmt.Errorf("readFirstPageInfoFromPagesObj: %w", err)
	}

	typ, ok := DictGetName(pagesDict, "Type")
	if !ok {
		return nil, fmt.Errorf("/Type key not found in pages object")
	}
	if typ != "Pages" {
		return nil, fmt.Errorf("object %s is /Type /%s, expected /Type /Pages", RefToString(pagesRefVal), typ)
	}

	kidsArr, ok := DictGetArray(pagesDict, "Kids")
	if !ok {
		return nil, fmt.Errorf("/Kids not found in pages object %s", RefToString(pagesRefVal))
	}

	refs := ArrayToRefs(kidsArr)
	if len(refs) == 0 {
		return nil, fmt.Errorf("pages object %s has empty /Kids", RefToString(pagesRefVal))
	}

	for _, ref := range refs {
		refStr := RefToString(ref)
		kidObj, err := readObject(f, refStr)
		if err != nil {
			continue
		}

		kidDict, err := extractDictFromObject(kidObj)
		if err != nil {
			continue
		}

		kidType, ok := DictGetName(kidDict, "Type")
		if !ok {
			continue
		}

		if kidType == "Pages" {
			info, err := readFirstPageInfoFromPagesObj(f, kidObj, ref)
			if err == nil {
				return info, nil
			}
			continue
		}

		if kidType == "Page" {
			parent, ok := DictGetRef(kidDict, "Parent")
			if !ok {
				return nil, fmt.Errorf("first /Type /Page object %s: /Parent key not found or not an indirect ref", refStr)
			}

			mediaBox, ok := DictGetArray(kidDict, "MediaBox")
			if !ok {
				mediaBox, err = lookupMediaBoxFromAncestors(refStr, f)
				if err != nil {
					return nil, fmt.Errorf("first /Type /Page object %s: %w", refStr, err)
				}
			}

			resources, ok := DictGetRef(kidDict, "Resources")
			inlineResources, hasInline := DictGetDict(kidDict, "Resources")
			if !ok && !hasInline {
				resources, err = lookupResourcesFromAncestors(refStr, f)
				if err != nil {
					return nil, fmt.Errorf("first /Type /Page object %s: %w", refStr, err)
				}
			}

			contents, err := readPageContentsRefs(kidDict)
			if err != nil {
				return nil, fmt.Errorf("first /Type /Page object %s: %w", refStr, err)
			}

			rotate, err := lookupRotateFromAncestors(refStr, f)
			if err != nil {
				return nil, fmt.Errorf("first /Type /Page object %s: %w", refStr, err)
			}

			info := &FirstPageInfo{
				PagesRef:        pagesRefVal,
				PageRef:         ref,
				Parent:          parent,
				MediaBox:        mediaBox,
				Resources:       resources,
				InlineResources: inlineResources,
				Contents:        contents,
				Rotate:          rotate,
			}
			return info, nil
		}
	}

	return nil, fmt.Errorf("no /Type /Page found in pages tree under %s", RefToString(pagesRefVal))
}

// lookupMediaBoxFromAncestors searches upward through the page tree for an
// inherited /MediaBox entry. It is a wrapper that enforces both a recursion
// depth limit and cycle detection via a visited map.
func lookupMediaBoxFromAncestors(pageRefStr string, f *os.File) (PDFArray, error) {
	return lookupMediaBoxFromAncestorsDepth(pageRefStr, f, 0, map[string]struct{}{})
}

// lookupMediaBoxFromAncestorsDepth is the internal recursive implementation.
// depth tracks recursion depth against maxAncestorLookupDepth; visited
// records page references already seen to detect cyclic /Parent chains.
func lookupMediaBoxFromAncestorsDepth(pageRefStr string, f *os.File, depth int, visited map[string]struct{}) (PDFArray, error) {
	if depth > maxAncestorLookupDepth {
		return nil, fmt.Errorf("lookupMediaBoxFromAncestors: ancestor lookup depth exceeds maximum %d", maxAncestorLookupDepth)
	}
	if _, seen := visited[pageRefStr]; seen {
		return nil, fmt.Errorf("lookupMediaBoxFromAncestors: cyclic /Parent reference detected at %s", pageRefStr)
	}
	visited[pageRefStr] = struct{}{}

	objStr, err := readObject(f, pageRefStr)
	if err != nil {
		return nil, fmt.Errorf("lookupMediaBoxFromAncestors: %w", err)
	}

	dict, err := extractDictFromObject(objStr)
	if err != nil {
		return nil, fmt.Errorf("lookupMediaBoxFromAncestors: %w", err)
	}

	typ, ok := DictGetName(dict, "Type")
	if !ok {
		return nil, fmt.Errorf("lookupMediaBoxFromAncestors: /Type key not found")
	}

	if typ == "Page" {
		if mb, ok := DictGetArray(dict, "MediaBox"); ok {
			return mb, nil
		}
		parent, ok := DictGetRef(dict, "Parent")
		if !ok {
			return nil, fmt.Errorf("lookupMediaBoxFromAncestors: /MediaBox not found and no /Parent in Page")
		}
		return lookupMediaBoxFromAncestorsDepth(RefToString(parent), f, depth+1, visited)
	}

	if typ == "Pages" {
		if mb, ok := DictGetArray(dict, "MediaBox"); ok {
			return mb, nil
		}
		parent, ok := DictGetRef(dict, "Parent")
		if !ok {
			return nil, fmt.Errorf("lookupMediaBoxFromAncestors: /MediaBox not found and no /Parent in Pages")
		}
		return lookupMediaBoxFromAncestorsDepth(RefToString(parent), f, depth+1, visited)
	}

	return nil, fmt.Errorf("lookupMediaBoxFromAncestors: object is /Type /%s, expected /Page or /Pages", typ)
}

// lookupResourcesFromAncestors searches upward through the page tree for an
// inherited /Resources entry. It is a wrapper that enforces both a recursion
// depth limit and cycle detection via a visited map.
func lookupResourcesFromAncestors(pageRefStr string, f *os.File) (PDFRef, error) {
	return lookupResourcesFromAncestorsDepth(pageRefStr, f, 0, map[string]struct{}{})
}

// lookupResourcesFromAncestorsDepth is the internal recursive implementation.
// depth tracks recursion depth against maxAncestorLookupDepth; visited
// records page references already seen to detect cyclic /Parent chains.
func lookupResourcesFromAncestorsDepth(pageRefStr string, f *os.File, depth int, visited map[string]struct{}) (PDFRef, error) {
	if depth > maxAncestorLookupDepth {
		return PDFRef{}, fmt.Errorf("lookupResourcesFromAncestors: ancestor lookup depth exceeds maximum %d", maxAncestorLookupDepth)
	}
	if _, seen := visited[pageRefStr]; seen {
		return PDFRef{}, fmt.Errorf("lookupResourcesFromAncestors: cyclic /Parent reference detected at %s", pageRefStr)
	}
	visited[pageRefStr] = struct{}{}

	objStr, err := readObject(f, pageRefStr)
	if err != nil {
		return PDFRef{}, fmt.Errorf("lookupResourcesFromAncestors: %w", err)
	}

	dict, err := extractDictFromObject(objStr)
	if err != nil {
		return PDFRef{}, fmt.Errorf("lookupResourcesFromAncestors: %w", err)
	}

	typ, ok := DictGetName(dict, "Type")
	if !ok {
		return PDFRef{}, fmt.Errorf("lookupResourcesFromAncestors: /Type key not found")
	}

	if typ == "Page" {
		if res, ok := DictGetRef(dict, "Resources"); ok {
			return res, nil
		}
		if _, hasInline := dict[PDFName("Resources")]; hasInline {
			return PDFRef{}, nil
		}
		parent, ok := DictGetRef(dict, "Parent")
		if !ok {
			return PDFRef{}, fmt.Errorf("lookupResourcesFromAncestors: /Resources not found and no /Parent in Page")
		}
		return lookupResourcesFromAncestorsDepth(RefToString(parent), f, depth+1, visited)
	}

	if typ == "Pages" {
		if res, ok := DictGetRef(dict, "Resources"); ok {
			return res, nil
		}
		if _, hasInline := dict[PDFName("Resources")]; hasInline {
			return PDFRef{}, nil
		}
		parent, ok := DictGetRef(dict, "Parent")
		if !ok {
			return PDFRef{}, fmt.Errorf("lookupResourcesFromAncestors: /Resources not found and no /Parent in Pages")
		}
		return lookupResourcesFromAncestorsDepth(RefToString(parent), f, depth+1, visited)
	}

	return PDFRef{}, fmt.Errorf("lookupResourcesFromAncestors: object is /Type /%s, expected /Page or /Pages", typ)
}

// lookupRotateFromAncestors searches upward through the page tree for an
// inherited /Rotate entry. It is a wrapper that enforces both a recursion
// depth limit and cycle detection via a visited map.
func lookupRotateFromAncestors(pageRefStr string, f *os.File) (*int64, error) {
	return lookupRotateFromAncestorsDepth(pageRefStr, f, 0, map[string]struct{}{})
}

// lookupRotateFromAncestorsDepth is the internal recursive implementation.
// depth tracks recursion depth against maxAncestorLookupDepth; visited
// records page references already seen to detect cyclic /Parent chains.
func lookupRotateFromAncestorsDepth(pageRefStr string, f *os.File, depth int, visited map[string]struct{}) (*int64, error) {
	if depth > maxAncestorLookupDepth {
		return nil, fmt.Errorf("lookupRotateFromAncestors: ancestor lookup depth exceeds maximum %d", maxAncestorLookupDepth)
	}
	if _, seen := visited[pageRefStr]; seen {
		return nil, fmt.Errorf("lookupRotateFromAncestors: cyclic /Parent reference detected at %s", pageRefStr)
	}
	visited[pageRefStr] = struct{}{}

	objStr, err := readObject(f, pageRefStr)
	if err != nil {
		return nil, fmt.Errorf("lookupRotateFromAncestors: %w", err)
	}

	dict, err := extractDictFromObject(objStr)
	if err != nil {
		return nil, fmt.Errorf("lookupRotateFromAncestors: %w", err)
	}

	typ, ok := DictGetName(dict, "Type")
	if !ok {
		return nil, fmt.Errorf("lookupRotateFromAncestors: /Type key not found")
	}

	if typ == "Page" {
		if obj := DictGet(dict, "Rotate"); obj != nil {
			rot, ok := obj.(PDFInteger)
			if !ok {
				return nil, fmt.Errorf("lookupRotateFromAncestors: /Rotate is not an integer")
			}
			v := int64(rot)
			return &v, nil
		}
		parent, ok := DictGetRef(dict, "Parent")
		if !ok {
			return nil, nil
		}
		return lookupRotateFromAncestorsDepth(RefToString(parent), f, depth+1, visited)
	}

	if typ == "Pages" {
		if obj := DictGet(dict, "Rotate"); obj != nil {
			rot, ok := obj.(PDFInteger)
			if !ok {
				return nil, fmt.Errorf("lookupRotateFromAncestors: /Rotate is not an integer")
			}
			v := int64(rot)
			return &v, nil
		}
		parent, ok := DictGetRef(dict, "Parent")
		if !ok {
			return nil, nil
		}
		return lookupRotateFromAncestorsDepth(RefToString(parent), f, depth+1, visited)
	}

	return nil, fmt.Errorf("lookupRotateFromAncestors: object is /Type /%s, expected /Page or /Pages", typ)
}

// ReadFirstPageInfo parses the PDF file to extract first-page structure from the Pages tree.
func ReadFirstPageInfo(f *os.File) (*FirstPageInfo, error) {
	xrefOffset, err := readStartxref(f)
	if err != nil {
		return nil, fmt.Errorf("ReadFirstPageInfo: %w", err)
	}

	rootRefStr, err := readTrailerRootRef(f, xrefOffset)
	if err != nil {
		return nil, fmt.Errorf("ReadFirstPageInfo: %w", err)
	}

	catalogObj, err := readObject(f, rootRefStr)
	if err != nil {
		return nil, fmt.Errorf("ReadFirstPageInfo: %w", err)
	}

	pagesRefStr, err := readPagesRefFromCatalog(catalogObj)
	if err != nil {
		return nil, fmt.Errorf("ReadFirstPageInfo: %w", err)
	}

	pagesObj, err := readObject(f, pagesRefStr)
	if err != nil {
		return nil, fmt.Errorf("ReadFirstPageInfo: %w", err)
	}

	pagesRefVal := parseRefString(pagesRefStr)

	return readFirstPageInfoFromPagesObj(f, pagesObj, pagesRefVal)
}

func parseRefString(refStr string) PDFRef {
	parts := strings.Fields(refStr)
	if len(parts) >= 2 {
		objNum, _ := strconv.ParseInt(parts[0], 10, 64)
		genNum, _ := strconv.ParseInt(parts[1], 10, 64)
		return PDFRef{ObjNum: objNum, GenNum: genNum}
	}
	return PDFRef{}
}

// ReadPageCount returns the declared total page count from the PDF page tree root.
// It reads the /Count entry from the root /Pages dictionary referenced by the Catalog.
// Returns 0 and an error if the count cannot be determined.
func ReadPageCount(f *os.File) (int, error) {
	xrefOffset, err := readStartxref(f)
	if err != nil {
		return 0, fmt.Errorf("ReadPageCount: %w", err)
	}

	rootRefStr, err := readTrailerRootRef(f, xrefOffset)
	if err != nil {
		return 0, fmt.Errorf("ReadPageCount: %w", err)
	}

	catalogObj, err := readObject(f, rootRefStr)
	if err != nil {
		return 0, fmt.Errorf("ReadPageCount: %w", err)
	}

	pagesRefStr, err := readPagesRefFromCatalog(catalogObj)
	if err != nil {
		return 0, fmt.Errorf("ReadPageCount: %w", err)
	}

	pagesObj, err := readObject(f, pagesRefStr)
	if err != nil {
		return 0, fmt.Errorf("ReadPageCount: %w", err)
	}

	d, err := extractDictFromObject(pagesObj)
	if err != nil {
		return 0, fmt.Errorf("ReadPageCount: %w", err)
	}

	count, ok := DictGetInt(d, "Count")
	if !ok {
		return 0, fmt.Errorf("ReadPageCount: /Count not found in Pages dict")
	}

	return int(count), nil
}
