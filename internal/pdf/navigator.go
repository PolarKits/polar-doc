package pdf

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/PolarKits/polar-doc/internal/doc"
)

// pdfNavigator provides random-access to PDF page content via object references.
// It can resolve a reference string like "12 0 R" to the page's content stream.
type pdfNavigator struct {
	doc *document
}

// newPDFNavigator creates a navigator for the given PDF document.
func newPDFNavigator(pdfDoc *document) *pdfNavigator {
	return &pdfNavigator{doc: pdfDoc}
}

// GoTo resolves a PDF object reference string and returns the page's content.
// The ref format must be "objNum genNum R" (e.g., "12 0 R").
// If the referenced object is a Page, its content stream is returned.
// If the referenced object is a Pages node, an error is returned.
func (n *pdfNavigator) GoTo(ctx context.Context, ref string) (doc.PageData, error) {
	select {
	case <-ctx.Done():
		return doc.PageData{}, ctx.Err()
	default:
	}

	f := n.doc.getFile()
	if f == nil {
		return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: document file is closed")
	}

	// Parse ref string: "objNum genNum R"
	parts := strings.Fields(ref)
	if len(parts) != 3 || parts[2] != "R" {
		return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: invalid ref format %q, expected \"objNum genNum R\"", ref)
	}

	objNum, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: parse objNum: %w", err)
	}
	genNum, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: parse genNum: %w", err)
	}

	pdfRef := PDFRef{ObjNum: objNum, GenNum: genNum}

	// Read the referenced object
	objStr, err := readObject(f, RefToString(pdfRef))
	if err != nil {
		return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: readObject: %w", err)
	}

	objDict, err := extractDictFromObject(objStr)
	if err != nil {
		return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: extractDict: %w", err)
	}

	typ, ok := DictGetName(objDict, "Type")
	if !ok {
		return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: /Type not found in object %s", ref)
	}

	if typ != "Page" {
		return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: ref %s is /Type /%s, expected /Type /Page", ref, typ)
	}

	// Collect MediaBox
	mediaBox, ok := DictGetArray(objDict, "MediaBox")
	if !ok {
		mediaBox, err = lookupMediaBoxFromAncestors(ref, f)
		if err != nil {
			return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: lookupMediaBox: %w", err)
		}
	}

	mb := make([]float64, len(mediaBox))
	for i, v := range mediaBox {
		switch x := v.(type) {
		case PDFReal:
			mb[i] = float64(x)
		case PDFInteger:
			mb[i] = float64(x)
		}
	}

	// Read page content streams
	contents, err := readPageContentsRefs(objDict)
	if err != nil {
		return doc.PageData{}, fmt.Errorf("pdfNavigator.GoTo: readPageContentsRefs: %w", err)
	}

	var contentBytes []byte
	for _, contentRef := range contents {
		streamData, err := readContentStream(f, contentRef)
		if err != nil {
			continue
		}
		contentBytes = append(contentBytes, streamData...)
		contentBytes = append(contentBytes, '\n')
	}

	// Determine 1-based page number by traversing Pages tree
	pageNum := n.findPageNumber(context.Background(), pdfRef)

	return doc.PageData{
		Number:   pageNum,
		ObjRef:   ref,
		MediaBox: mb,
		Content:  contentBytes,
	}, nil
}

// findPageNumber traverses the Pages tree to find the 1-based index of the given page ref.
func (n *pdfNavigator) findPageNumber(ctx context.Context, target PDFRef) int {
	f := n.doc.getFile()
	if f == nil {
		return 0
	}

	xrefOffset, err := readStartxref(f)
	if err != nil {
		return 0
	}

	rootRefStr, err := readTrailerRootRef(f, xrefOffset)
	if err != nil {
		return 0
	}

	catalogObj, err := readObject(f, rootRefStr)
	if err != nil {
		return 0
	}

	pagesRefStr, err := readPagesRefFromCatalog(catalogObj)
	if err != nil {
		return 0
	}

	pageNum := 0
	n.traverseForNumber(pagesRefStr, target, &pageNum, 0, make(map[string]struct{}))
	return pageNum
}

// traverseForNumber recursively searches the Pages tree for target and counts page numbers.
func (n *pdfNavigator) traverseForNumber(pagesRefStr string, target PDFRef, count *int, depth int, visited map[string]struct{}) {
	if depth > maxPageTreeDepth {
		return
	}
	if _, seen := visited[pagesRefStr]; seen {
		return
	}
	visited[pagesRefStr] = struct{}{}

	f := n.doc.getFile()
	objStr, err := readObject(f, pagesRefStr)
	if err != nil {
		return
	}

	objDict, err := extractDictFromObject(objStr)
	if err != nil {
		return
	}

	typ, ok := DictGetName(objDict, "Type")
	if !ok {
		return
	}

	if typ != "Pages" {
		return
	}

	kidsArr, ok := DictGetArray(objDict, "Kids")
	if !ok {
		return
	}

	refs := ArrayToRefs(kidsArr)
	for _, ref := range refs {
		refStr := RefToString(ref)
		if _, seen := visited[refStr]; seen {
			continue
		}

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
			n.traverseForNumber(refStr, target, count, depth+1, visited)
			continue
		}

		if kidType == "Page" {
			*count++
			if ref.ObjNum == target.ObjNum && ref.GenNum == target.GenNum {
				return
			}
		}
	}
}
