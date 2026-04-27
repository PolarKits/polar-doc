package pdf

import (
	"strings"
)

// parseXMPMetadata extracts basic metadata fields from XMP XML data.
// It performs simple XML tag matching without full XMP parsing.
// Returns a map of field name to value for: title, author, creator, producer, creation date, modification date.
func parseXMPMetadata(xmpData []byte) map[string]string {
	result := make(map[string]string)

	if len(xmpData) == 0 {
		return result
	}

	xmpStr := string(xmpData)

	fields := []struct {
		tag  string
		name string
	}{
		{"<dc:title>", "title"},
		{"<dc:creator>", "creator"},
		{"<xmp:CreatorTool>", "creator_tool"},
		{"<pdf:Producer>", "producer"},
		{"<xmp:CreateDate>", "creation_date"},
		{"<xmp:ModifyDate>", "modification_date"},
		{"<dc:subject>", "subject"},
		{"<dc:description>", "description"},
	}

	for _, field := range fields {
		start := strings.Index(xmpStr, field.tag)
		if start < 0 {
			continue
		}

		start += len(field.tag)

		endTag := "</" + field.tag[1:strings.Index(field.tag, ">")]
		endIdx := strings.Index(xmpStr[start:], endTag)
		if endIdx < 0 {
			continue
		}

		value := strings.TrimSpace(xmpStr[start : start+endIdx])
		value = extractXMLValueSimple(value)

		if value != "" {
			switch field.name {
			case "creator_tool":
				if _, ok := result["creator"]; !ok {
					result["creator"] = value
				}
			case "subject", "description":
				if _, ok := result["title"]; !ok {
					result["title"] = value
				}
			default:
				result[field.name] = value
			}
		}
	}

	return result
}

// extractXMLValueSimple strips XML tags from a string to extract plain text content.
func extractXMLValueSimple(s string) string {
	s = strings.TrimSpace(s)

	for {
		ltIdx := strings.Index(s, "<")
		if ltIdx < 0 {
			break
		}
		gtIdx := strings.Index(s[ltIdx:], ">")
		if gtIdx < 0 {
			break
		}
		s = s[:ltIdx] + s[ltIdx+gtIdx+1:]
	}

	return strings.TrimSpace(s)
}