package pdf

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// FontInfo describes a resolved PDF font and its encoding.
type FontInfo struct {
	// Name is the font resource name (e.g., "F1", "F2").
	Name string
	// Subtype is the font type (e.g., "Type1", "TrueType", "CIDFontType2").
	Subtype string
	// BaseFont is the font name from the dictionary.
	BaseFont string
	// Encoding is the font's character encoding name (e.g., "WinAnsiEncoding").
	Encoding string
	// ToUnicodeRef is the reference to the ToUnicode CMap stream (if present).
	ToUnicodeRef PDFRef
	// ToUnicode is the parsed CMap (character code → Unicode string), if available.
	ToUnicode map[rune]string
}

// resolvePageFonts resolves all fonts referenced by a page's Resources dictionary.
// It traverses Catalog → Pages → Page → /Resources /Font to build a complete
// font name → FontInfo mapping.
func resolvePageFonts(f *os.File, xrefOffset int64, pagesRef string, pageIndex int) (map[string]FontInfo, error) {
	fonts := make(map[string]FontInfo)

	// Build xref index for object resolution
	idx, err := buildXRefIndex(f, xrefOffset)
	if err != nil {
		return fonts, fmt.Errorf("build xref index: %w", err)
	}

	// Find the target page in the pages tree
	pageObj, err := findPageObject(f, idx, pagesRef, pageIndex)
	if err != nil {
		return fonts, fmt.Errorf("find page object: %w", err)
	}

	// Extract page dictionary
	pageDict, err := extractDictFromObject(pageObj)
	if err != nil {
		return fonts, fmt.Errorf("extract page dict: %w", err)
	}

	// Get Resources dictionary
	resourcesVal, ok := pageDict[PDFName("Resources")]
	if !ok {
		// No resources, return empty font map
		return fonts, nil
	}

	var resourcesDict PDFDict
	switch v := resourcesVal.(type) {
	case PDFDict:
		resourcesDict = v
	case PDFRef:
		// Resolve the resources reference
		resourcesObj, err := resolveObject(f, idx, v.ObjNum)
		if err != nil {
			return fonts, fmt.Errorf("resolve resources: %w", err)
		}
		resourcesDict, err = extractDictFromObject(string(resourcesObj))
		if err != nil {
			return fonts, fmt.Errorf("extract resources dict: %w", err)
		}
	default:
		return fonts, nil
	}

	// Get Font dictionary from Resources
	fontVal, ok := resourcesDict[PDFName("Font")]
	if !ok {
		// No fonts in resources
		return fonts, nil
	}

	var fontDict PDFDict
	switch v := fontVal.(type) {
	case PDFDict:
		fontDict = v
	case PDFRef:
		// Resolve the font dictionary reference
		fontObj, err := resolveObject(f, idx, v.ObjNum)
		if err != nil {
			return fonts, fmt.Errorf("resolve font dict: %w", err)
		}
		fontDict, err = extractDictFromObject(string(fontObj))
		if err != nil {
			return fonts, fmt.Errorf("extract font dict: %w", err)
		}
	default:
		return fonts, nil
	}

	// Parse each font entry in the Font dictionary
	for key, val := range fontDict {
		fontName := string(key)
		if !strings.HasPrefix(fontName, "F") && !strings.HasPrefix(fontName, "f") {
			// Font names typically start with F (e.g., /F1, /F2)
			// but some PDFs may use other naming conventions
			continue
		}
		// Remove leading / if present
		fontName = strings.TrimPrefix(fontName, "/")

		var fontInfo FontInfo
		fontInfo.Name = fontName

		switch v := val.(type) {
		case PDFRef:
			fontInfo = resolveFontInfo(f, idx, v, fontName)
		case PDFDict:
			fontInfo = parseFontDict(v, fontName)
		}

		// Try to load ToUnicode CMap if available
		if fontInfo.ToUnicodeRef.ObjNum != 0 {
			cmap, err := parseToUnicodeCMap(f, idx, fontInfo.ToUnicodeRef)
			if err == nil {
				fontInfo.ToUnicode = cmap
			}
		}

		fonts[fontName] = fontInfo
	}

	return fonts, nil
}

// resolveFontInfo resolves font information from a font object reference.
func resolveFontInfo(f *os.File, idx xrefIndex, ref PDFRef, name string) FontInfo {
	var info FontInfo
	info.Name = name

	fontObj, err := resolveObject(f, idx, ref.ObjNum)
	if err != nil {
		return info
	}

	fontDict, err := extractDictFromObject(string(fontObj))
	if err != nil {
		return info
	}

	return parseFontDict(fontDict, name)
}

// parseFontDict extracts font information from a font dictionary.
func parseFontDict(dict PDFDict, name string) FontInfo {
	info := FontInfo{Name: name}

	// Get Subtype
	if subtype, ok := DictGetName(dict, "Subtype"); ok {
		info.Subtype = string(subtype)
	}

	// Get BaseFont
	if baseFont, ok := DictGetName(dict, "BaseFont"); ok {
		info.BaseFont = string(baseFont)
	}

	// Get Encoding
	if encoding, ok := DictGetName(dict, "Encoding"); ok {
		info.Encoding = string(encoding)
	} else if encodingRef, ok := DictGetRef(dict, "Encoding"); ok && encodingRef.ObjNum != 0 {
		// Encoding is a reference, mark it for later resolution
		info.Encoding = fmt.Sprintf("%d %d R", encodingRef.ObjNum, encodingRef.GenNum)
	}

	// Get ToUnicode reference
	if toUnicodeRef, ok := DictGetRef(dict, "ToUnicode"); ok {
		info.ToUnicodeRef = toUnicodeRef
	}

	return info
}

// findPageObject finds a specific page object in the pages tree by index.
func findPageObject(f *os.File, idx xrefIndex, pagesRef string, targetIndex int) (string, error) {
	// Parse pages reference
	parts := strings.Fields(pagesRef)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid pages reference: %s", pagesRef)
	}

	pagesObjNum, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return "", fmt.Errorf("parse pages object number: %w", err)
	}

	// Read the Pages object
	pagesObj, err := resolveObject(f, idx, pagesObjNum)
	if err != nil {
		return "", fmt.Errorf("resolve pages object: %w", err)
	}

	pagesDict, err := extractDictFromObject(string(pagesObj))
	if err != nil {
		return "", fmt.Errorf("extract pages dict: %w", err)
	}

	// Check if this is a Pages node or a Page node
	if typ, ok := DictGetName(pagesDict, "Type"); ok && typ == "Page" {
		// This is already a page, return it if index matches
		if targetIndex == 0 {
			return string(pagesObj), nil
		}
		return "", fmt.Errorf("page index %d not found in single page", targetIndex)
	}

	// Get Kids array
	kidsVal, ok := pagesDict[PDFName("Kids")]
	if !ok {
		return "", fmt.Errorf("no Kids array in Pages")
	}

	kidsArr, ok := kidsVal.(PDFArray)
	if !ok {
		return "", fmt.Errorf("Kids is not an array")
	}

	// Find the target page by walking the tree
	currentIndex := 0
	for _, kidVal := range kidsArr {
		kidRef, ok := kidVal.(PDFRef)
		if !ok {
			continue
		}

		kidObj, err := resolveObject(f, idx, kidRef.ObjNum)
		if err != nil {
			continue
		}

		kidDict, err := extractDictFromObject(string(kidObj))
		if err != nil {
			continue
		}

		// Check if this is a Page or another Pages node
		if typ, ok := DictGetName(kidDict, "Type"); ok {
			if typ == "Page" {
				if currentIndex == targetIndex {
					return string(kidObj), nil
				}
				currentIndex++
			} else if typ == "Pages" {
				// This is a subtree, check its Count
				if countVal, ok := DictGetInt(kidDict, "Count"); ok {
					if currentIndex+int(countVal) > targetIndex {
						// The target is in this subtree
						kidRefStr := fmt.Sprintf("%d %d R", kidRef.ObjNum, kidRef.GenNum)
						return findPageObject(f, idx, kidRefStr, targetIndex-currentIndex)
					}
					currentIndex += int(countVal)
				}
			}
		}
	}

	return "", fmt.Errorf("page index %d not found", targetIndex)
}

// parseToUnicodeCMap parses a ToUnicode CMap stream and returns a character code to Unicode mapping.
// It handles both plain text and compressed (FlateDecode) CMap streams.
func parseToUnicodeCMap(f *os.File, idx xrefIndex, ref PDFRef) (map[rune]string, error) {
	cmap := make(map[rune]string)

	// Read the CMap object (includes dictionary and stream)
	cmapData, err := resolveObject(f, idx, ref.ObjNum)
	if err != nil {
		return cmap, fmt.Errorf("resolve CMap object: %w", err)
	}

	cmapStr := string(cmapData)

	// Parse the stream dictionary to get filter information
	// The dictionary is between the object header and "stream"
	streamKeyword := "stream"
	streamStart := strings.Index(cmapStr, streamKeyword)
	if streamStart < 0 {
		return cmap, fmt.Errorf("no stream keyword found")
	}

	// Extract dictionary portion (from << to before stream)
	dictStart := strings.Index(cmapStr, "<<")
	if dictStart < 0 || dictStart > streamStart {
		dictStart = 0
	}
	dictStr := cmapStr[dictStart:streamStart]

	// Parse filters from dictionary
	filters := parseFilterNames(dictStr)

	streamEnd := strings.Index(cmapStr, "endstream")
	if streamEnd < 0 {
		return cmap, fmt.Errorf("no endstream keyword found")
	}

	// Skip past "stream" and any newline to get raw stream bytes
	contentStart := streamStart + len(streamKeyword)
	for contentStart < streamEnd && (cmapStr[contentStart] == '\n' || cmapStr[contentStart] == '\r') {
		contentStart++
	}

	rawStream := cmapData[contentStart:streamEnd]

	// Decode the stream using stream_filter
	var decoded string
	if len(filters) > 0 {
		decodedBytes, err := decodeStream(rawStream, filters)
		if err != nil {
			// Decompression failed, try to use raw data as plain text
			decoded = string(rawStream)
		} else {
			decoded = string(decodedBytes)
		}
	} else {
		// No filters, assume plain text
		decoded = string(rawStream)
	}

	// Parse bfchar blocks
	bfcharRegex := regexp.MustCompile(`(\d+)\s+beginbfchar\s+(.*?)\s+endbfchar`)
	bfcharMatches := bfcharRegex.FindAllStringSubmatch(decoded, -1)
	for _, match := range bfcharMatches {
		// match[1] is the count, match[2] is the content
		content := match[2]
		// Parse character mappings in the content
		// Format: <XXXX> <YYYY> or <XXXX> [<YYYY ZZZZ>]
		mappingRegex := regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
		mappingMatches := mappingRegex.FindAllStringSubmatch(content, -1)
		for _, mapping := range mappingMatches {
			srcCode, _ := strconv.ParseInt(mapping[1], 16, 32)
			dstUnicode, _ := strconv.ParseInt(mapping[2], 16, 32)
			cmap[rune(srcCode)] = string(rune(dstUnicode))
		}
	}

	// Parse bfrange blocks (simplified - single unicode target only)
	bfrangeRegex := regexp.MustCompile(`(\d+)\s+beginbfrange\s+(.*?)\s+endbfrange`)
	bfrangeMatches := bfrangeRegex.FindAllStringSubmatch(decoded, -1)
	for _, match := range bfrangeMatches {
		content := match[2]
		// Format: <XXXX> <YYYY> <ZZZZ>  (start end target)
		rangeRegex := regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
		rangeMatches := rangeRegex.FindAllStringSubmatch(content, -1)
		for _, r := range rangeMatches {
			startCode, _ := strconv.ParseInt(r[1], 16, 32)
			endCode, _ := strconv.ParseInt(r[2], 16, 32)
			targetCode, _ := strconv.ParseInt(r[3], 16, 32)

			// Map the range
			for i := startCode; i <= endCode; i++ {
				offset := i - startCode
				cmap[rune(i)] = string(rune(targetCode + offset))
			}
		}
	}

	return cmap, nil
}
