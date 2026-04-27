package ofd

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// AnnotationType represents the type of an OFD annotation.
type AnnotationType string

const (
	AnnotationTypeHighlight AnnotationType = "Highlight"
	AnnotationTypeStamp     AnnotationType = "Stamp"
	AnnotationTypeWatermark AnnotationType = "Watermark"
	AnnotationTypeLink      AnnotationType = "Link"
	AnnotationTypeText      AnnotationType = "Text"
)

// Annotation represents a single annotation from OFD Annotations.xml.
type Annotation struct {
	ID        int64
	Type      AnnotationType
	Subtype   string
	PageID    int64
	Boundary  []float64
	Creator   string
	Modified  string
	Parameters map[string]string
}

// PageAnnotationIndex maps page IDs to their annotation file locations.
type PageAnnotationIndex struct {
	PageID   int64
	FilePath string
}

// AnnotationsDocument represents the parsed Annotations.xml structure.
type AnnotationsDocument struct {
	Pages []PageAnnotationIndex
}

// ParseAnnotationsXML reads Annotations.xml and returns its parsed content.
// Returns nil if the annotation file is not present in the package.
func ParseAnnotationsXML(files []*zip.File) (*AnnotationsDocument, error) {
	fileIndex := make(map[string]*zip.File, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	annotationsPath := "Doc_0/Annots/Annotations.xml"
	f, ok := fileIndex[annotationsPath]
	if !ok {
		return nil, nil
	}

	data, err := readFileContent(f)
	if err != nil {
		return nil, fmt.Errorf("read Annotations.xml: %w", err)
	}

	doc := &AnnotationsDocument{}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("xml token: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "Page" {
			pageIdx := PageAnnotationIndex{}
			for _, attr := range se.Attr {
				switch attr.Name.Local {
				case "PageID":
					if v, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
						pageIdx.PageID = v
					}
				}
			}
			for {
				tok, err := decoder.Token()
				if err != nil {
					break
				}
				if end, ok := tok.(xml.EndElement); ok && end.Name.Local == "Page" {
					break
				}
				if start, ok := tok.(xml.StartElement); ok && start.Name.Local == "FileLoc" {
					var content string
					if err := decoder.DecodeElement(&content, &start); err == nil {
						pageIdx.FilePath = strings.TrimSpace(content)
					}
				}
			}
			if pageIdx.PageID != 0 {
				doc.Pages = append(doc.Pages, pageIdx)
			}
		}
	}

	return doc, nil
}

// ParsePageAnnotations reads a single page annotation file (e.g., Page_0/Annotation.xml)
// and returns the list of annotations found.
func ParsePageAnnotations(data []byte) ([]Annotation, error) {
	var annotations []Annotation
	decoder := xml.NewDecoder(bytes.NewReader(data))

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("xml token: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "Annot" {
			annot := Annotation{Parameters: make(map[string]string)}
			for _, attr := range se.Attr {
				switch attr.Name.Local {
				case "ID":
					if v, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
						annot.ID = v
					}
				case "Type":
					annot.Type = AnnotationType(attr.Value)
				case "Subtype":
					annot.Subtype = attr.Value
				case "Creator":
					annot.Creator = attr.Value
				case "LastModDate":
					annot.Modified = attr.Value
				}
			}

			for {
				tok, err := decoder.Token()
				if err != nil {
					break
				}
				if end, ok := tok.(xml.EndElement); ok && end.Name.Local == "Annot" {
					break
				}

				switch v := tok.(type) {
				case xml.StartElement:
					local := v.Name.Local
					if local == "Appearance" {
						for _, attr := range v.Attr {
							if attr.Name.Local == "Boundary" {
								annot.Boundary = parseFloatArray(attr.Value)
							}
						}
					}
					if local == "Parameters" {
						for {
							tok, err := decoder.Token()
							if err != nil {
								break
							}
							if end, ok := tok.(xml.EndElement); ok && end.Name.Local == "Parameters" {
								break
							}
							if start, ok := tok.(xml.StartElement); ok && start.Name.Local == "Parameter" {
								var name, value string
								for _, attr := range start.Attr {
									if attr.Name.Local == "Name" {
										name = attr.Value
									}
								}
								var content string
								if err := decoder.DecodeElement(&content, &start); err == nil {
									value = content
								}
								if name != "" {
									annot.Parameters[name] = value
								}
							}
						}
					}
				}
			}

			annotations = append(annotations, annot)
		}
	}

	return annotations, nil
}