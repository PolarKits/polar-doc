package ofd

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/PolarKits/polar-doc/internal/doc"
)

// ResourceType represents the category of an OFD resource.
type ResourceType string

const (
	ResourceTypeFont      ResourceType = "Font"
	ResourceTypeMultiMedia ResourceType = "MultiMedia"
	ResourceTypeColorSpace ResourceType = "ColorSpace"
	ResourceTypeGraphicUnit ResourceType = "GraphicUnit"
)

// FontInfo holds parsed font metadata from a Resources.xml entry.
type FontInfo struct {
	ID         int64
	FontName   string
	FamilyName string
	Bold       bool
	FixedWidth bool
	CharSet    string
	FilePath   string
}

// MultiMediaInfo holds parsed multimedia metadata from a Resources.xml entry.
type MultiMediaInfo struct {
	ID       int64
	Type     string
	Format   string
	FilePath string
}

// DocumentResources holds the parsed content of a document-level Resources.xml.
// PublicRes.xml and DocumentRes.xml share the same structure; only their usage differs.
type DocumentResources struct {
	Fonts      []FontInfo
	MultiMedias []MultiMediaInfo
}

// ParseResourcesXML reads an OFD Resources.xml entry and returns its parsed content.
// It handles both PublicRes.xml and DocumentRes.xml since they share the same schema.
// Returns nil if the resource file is not present in the package.
func ParseResourcesXML(files []*zip.File) (*DocumentResources, error) {
	fileIndex := make(map[string]*zip.File, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	var candidates []string
	for _, name := range []string{"Doc_0/PublicRes.xml", "Doc_0/DocumentRes.xml"} {
		if _, ok := fileIndex[name]; ok {
			candidates = append(candidates, name)
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	result := &DocumentResources{}
	for _, name := range candidates {
		data, err := readFileContent(fileIndex[name])
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		if err := parseResourcesXMLData(data, result); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
	}

	if len(result.Fonts) == 0 && len(result.MultiMedias) == 0 {
		return nil, nil
	}
	return result, nil
}

func parseResourcesXMLData(data []byte, result *DocumentResources) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("xml token: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		local := se.Name.Local
		switch local {
		case "Font":
			font, err := parseFontElement(decoder, &se)
			if err != nil {
				return fmt.Errorf("parse font: %w", err)
			}
			if font != nil {
				result.Fonts = append(result.Fonts, *font)
			}
		case "MultiMedia":
			mm, err := parseMultiMediaElement(decoder, &se)
			if err != nil {
				return fmt.Errorf("parse multimedia: %w", err)
			}
			if mm != nil {
				result.MultiMedias = append(result.MultiMedias, *mm)
			}
		}
	}
	return nil
}

func parseFontElement(decoder *xml.Decoder, se *xml.StartElement) (*FontInfo, error) {
	font := &FontInfo{}

	for _, attr := range se.Attr {
		switch attr.Name.Local {
		case "ID":
			v, err := strconv.ParseInt(attr.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid Font ID: %w", err)
			}
			font.ID = v
		case "FontName":
			font.FontName = attr.Value
		case "FamilyName":
			font.FamilyName = attr.Value
		case "Bold":
			font.Bold = attr.Value == "true"
		case "FixedWidth":
			font.FixedWidth = attr.Value == "true"
		case "CharSet":
			font.CharSet = attr.Value
		}
	}

	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("font children: %w", err)
		}
		if tok == nil {
			break
		}

		switch v := tok.(type) {
		case xml.StartElement:
			if v.Name.Local == "FontFile" {
				var content string
				if err := decoder.DecodeElement(&content, &v); err != nil {
					return nil, fmt.Errorf("FontFile: %w", err)
				}
				font.FilePath = content
			}
		case xml.EndElement:
			if v.Name.Local == "Font" {
				return font, nil
			}
		}
	}

	return font, nil
}

func parseMultiMediaElement(decoder *xml.Decoder, se *xml.StartElement) (*MultiMediaInfo, error) {
	mm := &MultiMediaInfo{}

	for _, attr := range se.Attr {
		switch attr.Name.Local {
		case "ID":
			v, err := strconv.ParseInt(attr.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid MultiMedia ID: %w", err)
			}
			mm.ID = v
		case "Type":
			mm.Type = attr.Value
		case "Format":
			mm.Format = attr.Value
		}
	}

	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("multimedia children: %w", err)
		}
		if tok == nil {
			break
		}

		switch v := tok.(type) {
		case xml.StartElement:
			if v.Name.Local == "MediaFile" {
				var content string
				if err := decoder.DecodeElement(&content, &v); err != nil {
					return nil, fmt.Errorf("MediaFile: %w", err)
				}
				mm.FilePath = content
			}
		case xml.EndElement:
			if v.Name.Local == "MultiMedia" {
				return mm, nil
			}
		}
	}

	return mm, nil
}

// readFileContent reads and returns the full content of a zip file entry.
func readFileContent(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return data, nil
}

// GetDocument returns the OFD document from a generic doc.Document.
// It returns an error if the document is not an OFD document.
func GetDocument(d doc.Document) (*document, error) {
	ofdDoc, ok := d.(*document)
	if !ok {
		return nil, fmt.Errorf("not an OFD document: %T", d)
	}
	return ofdDoc, nil
}