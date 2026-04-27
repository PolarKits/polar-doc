package ofd

import (
	"encoding/xml"
	"strconv"
	"strings"
)

// TextCode represents a single text code element within a TextObject.
// It holds both the text content and positioning attributes.
type TextCode struct {
	Text    string
	X       float64
	Y       float64
	DeltaX  []float64
	DeltaY  []float64
	Size    float64
}

// TextObject represents a text object element from OFD Content.xml.
// TextObjects contain one or more TextCode elements and carry positioning
// and font resource information.
type TextObject struct {
	ID        int64
	Boundary  []float64 // [x, y, width, height]
	Font      int64
	Size      float64
	Codes     []TextCode
	Direction string
}

// parseTextObjects extracts all TextObject elements from Content.xml data.
// It handles TextObject/TextCode hierarchy and collects all positioning
// and content information without modifying existing text extraction.
func parseTextObjects(data []byte) ([]TextObject, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var objects []TextObject
	var currentObj *TextObject
	var inTextCode bool
	var charData *strings.Builder

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := tok.(type) {
		case xml.StartElement:
			local := strings.TrimPrefix(t.Name.Local, "ofd:")

			switch local {
			case "TextObject":
				obj := TextObject{}
				for _, attr := range t.Attr {
					attrName := strings.TrimPrefix(attr.Name.Local, "ofd:")
					switch attrName {
					case "ID":
						parseIntAttr(attr.Value, &obj.ID)
					case "Boundary":
						obj.Boundary = parseFloatArray(attr.Value)
					case "Font":
						parseIntAttr(attr.Value, &obj.Font)
					case "Size":
						parseFloatAttr(attr.Value, &obj.Size)
					case "Direction":
						obj.Direction = attr.Value
					}
				}
				currentObj = &obj
				charData = nil

			case "TextCode":
				inTextCode = true
				code := TextCode{Size: currentObj.Size}
				for _, attr := range t.Attr {
					attrName := strings.TrimPrefix(attr.Name.Local, "ofd:")
					switch attrName {
					case "X":
						parseFloatAttr(attr.Value, &code.X)
					case "Y":
						parseFloatAttr(attr.Value, &code.Y)
					case "Size":
						parseFloatAttr(attr.Value, &code.Size)
					case "DeltaX":
						code.DeltaX = parseFloatArray(attr.Value)
					case "DeltaY":
						code.DeltaY = parseFloatArray(attr.Value)
					}
				}
				if currentObj != nil {
					charData = new(strings.Builder)
					currentObj.Codes = append(currentObj.Codes, code)
				}

			case "Caret", "MarkedContent":
			}
		case xml.EndElement:
			local := strings.TrimPrefix(t.Name.Local, "ofd:")
			if local == "TextCode" {
				inTextCode = false
				if charData != nil && len(currentObj.Codes) > 0 {
					lastIdx := len(currentObj.Codes) - 1
					currentObj.Codes[lastIdx].Text = strings.TrimSpace(charData.String())
				}
				charData = nil
			}
			if local == "TextObject" && currentObj != nil {
				objects = append(objects, *currentObj)
				currentObj = nil
			}
		case xml.CharData:
			if inTextCode && charData != nil {
				charData.Write(t)
			}
		}
	}

	return objects, nil
}

func parseFloatAttr(val string, target *float64) {
	if v, err := strconv.ParseFloat(val, 64); err == nil {
		*target = v
	}
}

func parseIntAttr(val string, target *int64) {
	if v, err := strconv.ParseInt(val, 10, 64); err == nil {
		*target = v
	}
}

func parseFloatArray(val string) []float64 {
	parts := strings.Fields(val)
	result := make([]float64, 0, len(parts))
	for _, p := range parts {
		if p == "g" {
			continue
		}
		if v, err := strconv.ParseFloat(p, 64); err == nil {
			result = append(result, v)
		}
	}
	return result
}