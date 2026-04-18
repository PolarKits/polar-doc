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

	"github.com/PolarKits/polardoc/internal/doc"
)

type FirstPageInfo struct {
	PagesRef        PDFRef
	PageRef         PDFRef
	Parent          PDFRef
	MediaBox        PDFArray
	Resources       PDFRef
	InlineResources PDFDict
	Contents        []PDFRef
	Rotate          *int64
}

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

func (d *document) getFile() *os.File {
	return d.file
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

func (s *service) ExtractText(_ context.Context, d doc.Document) (doc.TextResult, error) {
	pdfDoc, ok := d.(*document)
	if !ok {
		return doc.TextResult{}, fmt.Errorf("unsupported document type %T", d)
	}

	_ = pdfDoc
	return doc.TextResult{}, fmt.Errorf("text extraction is not implemented for PDF")
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

func readObject(f *os.File, ref string) (string, error) {
	parts := strings.Fields(ref)
	if len(parts) < 3 || parts[2] != "R" {
		return "", fmt.Errorf("invalid object ref %q", ref)
	}
	objNum := parts[0]
	genNum := parts[1]

	xrefOffset, err := readStartxref(f)
	if err != nil {
		return "", fmt.Errorf("readStartxref for object lookup: %w", err)
	}

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

func readFirstPageFromPages(f *os.File, pagesRef string) (string, error) {
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
			page, err := readFirstPageFromPages(f, refStr)
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

	_, err := f.Seek(xrefOffset, io.SeekStart)
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

		if line == "trailer" || line == "xref" {
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
						return offset, nil
					}
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
					return offset, nil
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

	_, err = f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	objOffset, err := findObjectOffsetInXRefStream(f, xrefOffset, objNumInt)
	if err == nil {
		return objOffset, nil
	}

	offset, err := findObjectOffsetInFileBody(f, objNumInt)
	if err == nil {
		return offset, nil
	}

	return 0, fmt.Errorf("object %s not found in xref", objNum)
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

func lookupMediaBoxFromAncestors(pageRefStr string, f *os.File) (PDFArray, error) {
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
		return lookupMediaBoxFromAncestors(RefToString(parent), f)
	}

	if typ == "Pages" {
		if mb, ok := DictGetArray(dict, "MediaBox"); ok {
			return mb, nil
		}
		parent, ok := DictGetRef(dict, "Parent")
		if !ok {
			return nil, fmt.Errorf("lookupMediaBoxFromAncestors: /MediaBox not found and no /Parent in Pages")
		}
		return lookupMediaBoxFromAncestors(RefToString(parent), f)
	}

	return nil, fmt.Errorf("lookupMediaBoxFromAncestors: object is /Type /%s, expected /Page or /Pages", typ)
}

func lookupResourcesFromAncestors(pageRefStr string, f *os.File) (PDFRef, error) {
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
		return lookupResourcesFromAncestors(RefToString(parent), f)
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
		return lookupResourcesFromAncestors(RefToString(parent), f)
	}

	return PDFRef{}, fmt.Errorf("lookupResourcesFromAncestors: object is /Type /%s, expected /Page or /Pages", typ)
}

func lookupRotateFromAncestors(pageRefStr string, f *os.File) (*int64, error) {
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
		return lookupRotateFromAncestors(RefToString(parent), f)
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
		return lookupRotateFromAncestors(RefToString(parent), f)
	}

	return nil, fmt.Errorf("lookupRotateFromAncestors: object is /Type /%s, expected /Page or /Pages", typ)
}

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
