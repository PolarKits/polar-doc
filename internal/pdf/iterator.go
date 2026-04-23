package pdf

import (
	"context"
	"fmt"
	"io"

	"github.com/PolarKits/polar-doc/internal/doc"
)

// pageIterator provides sequential streaming access to PDF pages.
// It uses an explicit stack to traverse the Pages tree iteratively,
// avoiding deep recursion for documents with many nested Pages nodes.
type pageIterator struct {
	doc       *document
	pageNum   int                 // 1-based count of pages returned so far
	kidsStack [][]PDFRef           // stack of Kids arrays to process
	visited   map[string]struct{}  // cycle detection: seen page/kids refs
	pageCount int                  // total pages (from /Count in root Pages)
	pageData  doc.PageData         // lookahead: content of current page
	exhausted bool                 // true after Next returns io.EOF
}

// newPageIterator creates a pageIterator for the given document.
// It initializes the iterator to traverse from the root Pages node.
func newPageIterator(pdfDoc *document) (*pageIterator, error) {
	f := pdfDoc.getFile()
	if f == nil {
		return nil, fmt.Errorf("pageIterator: document file is not open")
	}

	xrefOffset, err := readStartxref(f)
	if err != nil {
		return nil, fmt.Errorf("pageIterator: readStartxref: %w", err)
	}

	rootRefStr, err := readTrailerRootRef(f, xrefOffset)
	if err != nil {
		return nil, fmt.Errorf("pageIterator: readTrailerRootRef: %w", err)
	}

	catalogObj, err := pdfDoc.readObject(rootRefStr)
	if err != nil {
		return nil, fmt.Errorf("pageIterator: readCatalog: %w", err)
	}

	pagesRefStr, err := readPagesRefFromCatalog(catalogObj)
	if err != nil {
		return nil, fmt.Errorf("pageIterator: readPagesRef: %w", err)
	}

	// Read root Pages to get /Count (total page count)
	pagesObj, err := pdfDoc.readObject(pagesRefStr)
	if err != nil {
		return nil, fmt.Errorf("pageIterator: readPages: %w", err)
	}
	pagesDict, err := extractDictFromObject(pagesObj)
	if err != nil {
		return nil, fmt.Errorf("pageIterator: extractPagesDict: %w", err)
	}
	count, _ := DictGetInt(pagesDict, "Count")

	// Get the root Kids array to seed the stack
	kidsArr, ok := DictGetArray(pagesDict, "Kids")
	if !ok {
		return nil, fmt.Errorf("pageIterator: /Kids not found in root Pages")
	}
	refs := ArrayToRefs(kidsArr)

	return &pageIterator{
		doc:       pdfDoc,
		kidsStack: [][]PDFRef{refs},
		visited:   make(map[string]struct{}),
		pageCount: int(count),
	}, nil
}

// Next returns the next page data or io.EOF when all pages have been visited.
// This method is not safe for concurrent use.
func (it *pageIterator) Next(ctx context.Context) (doc.PageData, error) {
	if it.exhausted {
		return doc.PageData{}, io.EOF
	}

	f := it.doc.getFile()
	if f == nil {
		return doc.PageData{}, fmt.Errorf("pageIterator.Next: document file is closed")
	}

	for len(it.kidsStack) > 0 {
		kids := it.kidsStack[len(it.kidsStack)-1]
		if len(kids) == 0 {
			// Pop empty array from stack
			it.kidsStack = it.kidsStack[:len(it.kidsStack)-1]
			continue
		}

		// Pop the first ref from the current Kids array
		ref := kids[0]
		it.kidsStack[len(it.kidsStack)-1] = kids[1:]

		refStr := RefToString(ref)
		if _, seen := it.visited[refStr]; seen {
			continue
		}
		it.visited[refStr] = struct{}{}

		objStr, err := it.doc.readObject(refStr)
		if err != nil {
			continue
		}

		objDict, err := extractDictFromObject(objStr)
		if err != nil {
			continue
		}

		typ, ok := DictGetName(objDict, "Type")
		if !ok {
			continue
		}

		if typ == "Pages" {
			// Descend into nested Pages node
			kidsArr, ok := DictGetArray(objDict, "Kids")
			if !ok {
				continue
			}
			refs := ArrayToRefs(kidsArr)
			it.kidsStack = append(it.kidsStack, refs)
			continue
		}

		if typ == "Page" {
			it.pageNum++

			// Collect MediaBox (with inheritance from ancestors handled by readAllPagesDepth logic)
			mediaBox, ok := DictGetArray(objDict, "MediaBox")
			if !ok {
				mediaBox, _ = lookupMediaBoxFromAncestors(refStr, f)
			}

			// Convert MediaBox to float64 slice for PageData
			mb := make([]float64, len(mediaBox))
			for i, v := range mediaBox {
				switch x := v.(type) {
				case PDFReal:
					mb[i] = float64(x)
				case PDFInteger:
					mb[i] = float64(x)
				}
			}

			// Read page content stream
			contents, err := readPageContentsRefs(objDict)
			if err != nil || len(contents) == 0 {
				continue
			}

			var contentBytes []byte
			for _, contentRef := range contents {
				streamData, err := it.doc.readCachedContentStream(contentRef)
				if err != nil {
					continue
				}
				contentBytes = append(contentBytes, streamData...)
				contentBytes = append(contentBytes, '\n')
			}

			return doc.PageData{
				Number:   it.pageNum,
				ObjRef:   refStr,
				MediaBox: mb,
				Content:  contentBytes,
			}, nil
		}
	}

	it.exhausted = true
	return doc.PageData{}, io.EOF
}

// Reset restarts the iterator from the first page.
// After Reset, the next call to Next returns the first page (Number=1).
func (it *pageIterator) Reset() {
	it.pageNum = 0
	it.exhausted = false

	// Re-read root Pages to get fresh Kids array
	f := it.doc.getFile()
	if f == nil {
		return
	}

	xrefOffset, err := readStartxref(f)
	if err != nil {
		return
	}

	rootRefStr, err := readTrailerRootRef(f, xrefOffset)
	if err != nil {
		return
	}

	catalogObj, err := it.doc.readObject(rootRefStr)
	if err != nil {
		return
	}

	pagesRefStr, err := readPagesRefFromCatalog(catalogObj)
	if err != nil {
		return
	}

	pagesObj, err := it.doc.readObject(pagesRefStr)
	if err != nil {
		return
	}

	pagesDict, err := extractDictFromObject(pagesObj)
	if err != nil {
		return
	}

	kidsArr, ok := DictGetArray(pagesDict, "Kids")
	if !ok {
		return
	}
	refs := ArrayToRefs(kidsArr)

	it.kidsStack = [][]PDFRef{refs}
	it.visited = make(map[string]struct{})
}

// PageCount returns the total number of pages as declared in the PDF.
func (it *pageIterator) PageCount(ctx context.Context) (int, error) {
	return it.pageCount, nil
}

// pdfPageIterator wraps pageIterator to implement doc.PageIterator.
type pdfPageIterator struct {
	iter *pageIterator
}

// Next implements doc.PageIterator.
func (p *pdfPageIterator) Next(ctx context.Context) (doc.PageData, error) {
	return p.iter.Next(ctx)
}

// Reset implements doc.PageIterator.
func (p *pdfPageIterator) Reset() {
	p.iter.Reset()
}
