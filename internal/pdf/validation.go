package pdf

import (
	"fmt"
	"io"
	"os"
	"regexp"

	"github.com/PolarKits/polar-doc/internal/doc"
)

// ValidationLevel represents a specific validation stage.
// Each level checks a different aspect of PDF structure.
type ValidationLevel int

const (
	// LevelHeader checks %PDF- prefix and version format.
	LevelHeader ValidationLevel = iota
	// LevelXRef checks startxref offset and xref table/stream integrity.
	LevelXRef
	// LevelTrailer checks trailer dictionary for required fields.
	LevelTrailer
	// LevelCatalog checks root catalog dictionary structure.
	LevelCatalog
	// LevelPages checks pages tree basic integrity.
	LevelPages
	// LevelFonts checks font resource reference integrity.
	LevelFonts
)

// LevelResult holds validation outcome for a single level.
type LevelResult struct {
	Level   ValidationLevel
	Passed  bool
	Errors  []string
	Warnings []string
}

// validateDocument performs comprehensive multi-level PDF validation.
// It collects errors and warnings from all levels to provide a complete report.
// Even if a level fails, subsequent levels are executed to gather maximum information.
func validateDocument(f *os.File) (doc.ValidationReport, error) {
	report := doc.ValidationReport{
		Valid: true,
	}

	// Level 1: Header check (fatal if failed)
	headerResult := validateHeader(f)
	report.Errors = append(report.Errors, headerResult.Errors...)
	report.Warnings = append(report.Warnings, headerResult.Warnings...)
	if !headerResult.Passed {
		report.Valid = false
		// Header is fatal - return early but still try other levels if possible
	}

	// Level 2: XRef structure check
	xrefResult := validateXRefStructure(f)
	report.Errors = append(report.Errors, xrefResult.Errors...)
	report.Warnings = append(report.Warnings, xrefResult.Warnings...)
	if !xrefResult.Passed {
		report.Valid = false
	}

	// Level 3: Trailer dictionary check
	trailerResult := validateTrailer(f)
	report.Errors = append(report.Errors, trailerResult.Errors...)
	report.Warnings = append(report.Warnings, trailerResult.Warnings...)
	if !trailerResult.Passed {
		report.Valid = false
	}

	// Level 4: Catalog check
	catalogResult := validateCatalog(f)
	report.Errors = append(report.Errors, catalogResult.Errors...)
	report.Warnings = append(report.Warnings, catalogResult.Warnings...)
	if !catalogResult.Passed {
		report.Valid = false
	}

	// Level 5: Pages tree check
	pagesResult := validatePages(f)
	report.Errors = append(report.Errors, pagesResult.Errors...)
	report.Warnings = append(report.Warnings, pagesResult.Warnings...)
	if !pagesResult.Passed {
		report.Valid = false
	}

	// Level 6: Fonts reference check
	fontsResult := validateFonts(f)
	report.Errors = append(report.Errors, fontsResult.Errors...)
	report.Warnings = append(report.Warnings, fontsResult.Warnings...)
	if !fontsResult.Passed {
		report.Valid = false
	}

	return report, nil
}

// validateHeader checks the PDF header for %PDF- prefix and valid version.
func validateHeader(f *os.File) LevelResult {
	result := LevelResult{Level: LevelHeader, Passed: true}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("header: seek error: %v", err))
		return result
	}

	version, err := readPDFHeaderVersion(f)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("header: %v", err))
		return result
	}

	// Validate version format X.Y
	if !isValidPDFVersion(version) {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("header: invalid version format: %s", version))
		return result
	}

	return result
}

// isValidPDFVersion checks if version string contains valid X.Y format.
// It extracts the first occurrence of a version pattern like "1.4" from the string.
// PDF version headers may contain additional binary data after the version number.
func isValidPDFVersion(version string) bool {
	// Extract version pattern from the string (handles cases like "1.4\r%...")
	re := regexp.MustCompile(`(\d+\.\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 2 {
		return false
	}
	// Validate the extracted version
	match, _ := regexp.MatchString(`^\d+\.\d+$`, matches[1])
	return match
}

// validateXRefStructure checks the xref table/stream integrity.
func validateXRefStructure(f *os.File) LevelResult {
	result := LevelResult{Level: LevelXRef, Passed: true}

	// Check startxref offset
	xrefOffset, err := readStartxref(f)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("xref: %v", err))
		return result
	}

	// Get file size for bounds checking
	info, err := f.Stat()
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("xref: stat error: %v", err))
		return result
	}

	if xrefOffset < 0 || xrefOffset > info.Size() {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("xref: startxref offset %d out of bounds", xrefOffset))
		return result
	}

	// Try to parse xref (traditional table or stream)
	idx, err := buildXRefIndex(f, xrefOffset)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("xref: build index: %v", err))
		return result
	}

	if len(idx) == 0 {
		result.Passed = false
		result.Errors = append(result.Errors, "xref: no objects found in xref index")
		return result
	}

	// Verify xref entries point to valid content
	for objNum, entry := range idx {
		if objNum == 0 {
			continue // Object 0 is special (free list head)
		}
		if entry.Kind == xrefEntryDirect {
			if entry.Offset < 0 || entry.Offset > info.Size() {
				result.Passed = false
				result.Errors = append(result.Errors, fmt.Sprintf("xref: object %d offset %d out of bounds", objNum, entry.Offset))
			}
		}
	}

	return result
}

// validateTrailer checks the trailer dictionary for required fields.
func validateTrailer(f *os.File) LevelResult {
	result := LevelResult{Level: LevelTrailer, Passed: true}

	// Get xref offset first
	xrefOffset, err := readStartxref(f)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("trailer: cannot find xref: %v", err))
		return result
	}

	// Read trailer dictionary
	trailerDict, isXRefStream, err := readTrailerDictFromFile(f, xrefOffset)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("trailer: read error: %v", err))
		return result
	}
	if trailerDict == "" {
		result.Passed = false
		result.Errors = append(result.Errors, "trailer: dictionary not found")
		return result
	}

	trailer, err := ParseDictContent(trailerDict)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("trailer: parse error: %v", err))
		return result
	}

	// Check /Root reference (required except in linearized PDFs which are rare)
	if _, ok := DictGetRef(trailer, "Root"); !ok && !isXRefStream {
		result.Passed = false
		result.Errors = append(result.Errors, "trailer: required /Root reference not found")
	}

	// Check /Size value (required, must be positive integer)
	if size, ok := DictGetInt(trailer, "Size"); ok {
		if size <= 0 {
			result.Passed = false
			result.Errors = append(result.Errors, fmt.Sprintf("trailer: /Size must be positive integer, got: %d", size))
		}
	} else {
		result.Passed = false
		result.Errors = append(result.Errors, "trailer: required /Size not found")
	}

	// Check /Encrypt (warning if present but algorithm unknown)
	if _, ok := trailer["Encrypt"]; ok {
		// Encryption present - would need deeper check for algorithm detection
		// For now, just note that encryption validation would happen here
		_ = ok
	}

	return result
}

// validateCatalog checks the root catalog dictionary structure.
func validateCatalog(f *os.File) LevelResult {
	result := LevelResult{Level: LevelCatalog, Passed: true}

	// Get xref offset
	xrefOffset, err := readStartxref(f)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("catalog: cannot find xref: %v", err))
		return result
	}

	// Get root reference
	rootRefStr, err := readTrailerRootRef(f, xrefOffset)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("catalog: cannot get root ref: %v", err))
		return result
	}

	// Read catalog object
	catalogObj, err := readObject(f, rootRefStr)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("catalog: cannot read catalog object: %v", err))
		return result
	}

	catalogDict, err := extractDictFromObject(catalogObj)
	if err != nil {
		// Catalog dict extraction failed - log as warning for backward compatibility
		result.Warnings = append(result.Warnings, fmt.Sprintf("catalog: cannot extract catalog dict: %v", err))
		return result
	}

	// Check /Type /Catalog
	if typ, ok := DictGetName(catalogDict, "Type"); !ok || typ != "Catalog" {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("catalog: expected /Type /Catalog, got /Type /%s", typ))
		return result
	}

	// Check /Pages reference exists
	if _, ok := DictGetRef(catalogDict, "Pages"); !ok {
		result.Passed = false
		result.Errors = append(result.Errors, "catalog: required /Pages reference not found")
		return result
	}

	return result
}

// validatePages checks the pages tree basic integrity.
func validatePages(f *os.File) LevelResult {
	result := LevelResult{Level: LevelPages, Passed: true}

	// Get xref offset
	xrefOffset, err := readStartxref(f)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("pages: cannot find xref: %v", err))
		return result
	}

	// Get root reference to find catalog
	rootRefStr, err := readTrailerRootRef(f, xrefOffset)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("pages: cannot get root ref: %v", err))
		return result
	}

	// Read catalog to get Pages reference
	catalogObj, err := readObject(f, rootRefStr)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("pages: cannot read catalog: %v", err))
		return result
	}

	catalogDict, err := extractDictFromObject(catalogObj)
	if err != nil {
		// Catalog dict extraction failed - log as warning for backward compatibility
		result.Warnings = append(result.Warnings, fmt.Sprintf("pages: cannot extract catalog dict: %v", err))
		return result
	}

	pagesRef, ok := DictGetRef(catalogDict, "Pages")
	if !ok {
		result.Passed = false
		result.Errors = append(result.Errors, "pages: /Pages reference not found in catalog")
		return result
	}

	// Read Pages object
	pagesRefStr := fmt.Sprintf("%d %d R", pagesRef.ObjNum, pagesRef.GenNum)
	pagesObj, err := readObject(f, pagesRefStr)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("pages: cannot read pages object: %v", err))
		return result
	}

	pagesDict, err := extractDictFromObject(pagesObj)
	if err != nil {
		// Pages dict extraction failed - log as warning for backward compatibility
		result.Warnings = append(result.Warnings, fmt.Sprintf("pages: cannot extract pages dict: %v", err))
		return result
	}

	// Check /Type /Pages
	if typ, ok := DictGetName(pagesDict, "Type"); !ok || typ != "Pages" {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("pages: expected /Type /Pages, got /Type /%s", typ))
		return result
	}

	// Check /Count (non-negative integer, warning if 0)
	if count, ok := DictGetInt(pagesDict, "Count"); ok {
		if count < 0 {
			result.Passed = false
			result.Errors = append(result.Errors, fmt.Sprintf("pages: /Count must be non-negative, got: %d", count))
		} else if count == 0 {
			// Count of 0 is unusual but may be valid in some edge cases
			result.Warnings = append(result.Warnings, "pages: /Count is 0, document may have no pages")
		}
	} else {
		// /Count is required, but we'll make it a warning to be lenient
		result.Warnings = append(result.Warnings, "pages: /Count not found")
	}

	// Check /Kids (array, warn if empty)
	kidsVal, ok := pagesDict[PDFName("Kids")]
	if !ok {
		result.Warnings = append(result.Warnings, "pages: /Kids not found")
		return result
	}

	kidsArr, ok := kidsVal.(PDFArray)
	if !ok {
		result.Passed = false
		result.Errors = append(result.Errors, "pages: /Kids must be an array")
		return result
	}

	if len(kidsArr) == 0 {
		result.Warnings = append(result.Warnings, "pages: /Kids array is empty")
	}

	// Check each individual Page dict for required fields.
	validatePageDictFields(f, pagesRefStr, &result, 0, map[string]struct{}{})

	return result
}

// validatePageDictFields recursively walks the Pages tree and checks each Page
// dict for required fields: /MediaBox (error if absent even after inheritance
// lookup) and /Resources (warning if absent).
func validatePageDictFields(f *os.File, pagesRef string, result *LevelResult, depth int, visited map[string]struct{}) {
	if depth > maxPageTreeDepth {
		return
	}
	if _, seen := visited[pagesRef]; seen {
		return
	}
	visited[pagesRef] = struct{}{}

	pagesObj, err := readObject(f, pagesRef)
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

	for _, kidRef := range ArrayToRefs(kidsArr) {
		refStr := RefToString(kidRef)
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
			validatePageDictFields(f, refStr, result, depth+1, visited)
			continue
		}

		if kidType != "Page" {
			continue
		}

		// Check /MediaBox: required either directly or via ancestor inheritance.
		_, hasMediaBox := DictGetArray(kidDict, "MediaBox")
		if !hasMediaBox {
			_, err := lookupMediaBoxFromAncestors(refStr, f)
			hasMediaBox = (err == nil)
		}
		if !hasMediaBox {
			result.Passed = false
			result.Errors = append(result.Errors, fmt.Sprintf("page %s: required /MediaBox not found (no inherited value)", refStr))
		}

		// Check /Resources: recommended; warn if absent even after inheritance lookup.
		_, hasRes := DictGetRef(kidDict, "Resources")
		_, hasInlineRes := DictGetDict(kidDict, "Resources")
		if !hasRes && !hasInlineRes {
			_, err := lookupResourcesFromAncestors(refStr, f)
			hasRes = (err == nil)
		}
		if !hasRes && !hasInlineRes {
			result.Warnings = append(result.Warnings, fmt.Sprintf("page %s: /Resources not found (no inherited value)", refStr))
		}
	}
}

// validateFonts checks font resource reference integrity for all pages.
// It traverses the Pages tree, extracts /Resources dictionaries, and validates
// each font entry has required fields (/Type, /Subtype, /BaseFont).
func validateFonts(f *os.File) LevelResult {
	result := LevelResult{Level: LevelFonts, Passed: true}

	xrefOffset, err := readStartxref(f)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("fonts: cannot find xref: %v", err))
		return result
	}

	rootRefStr, err := readTrailerRootRef(f, xrefOffset)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("fonts: cannot get root ref: %v", err))
		return result
	}

	catalogObj, err := readObject(f, rootRefStr)
	if err != nil {
		result.Passed = false
		result.Errors = append(result.Errors, fmt.Sprintf("fonts: cannot read catalog: %v", err))
		return result
	}

	catalogDict, err := extractDictFromObject(catalogObj)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("fonts: cannot extract catalog dict: %v", err))
		return result
	}

	pagesRef, ok := DictGetRef(catalogDict, "Pages")
	if !ok {
		result.Passed = false
		result.Errors = append(result.Errors, "fonts: /Pages reference not found in catalog")
		return result
	}

	pagesRefStr := fmt.Sprintf("%d %d R", pagesRef.ObjNum, pagesRef.GenNum)
	validateFontResources(f, pagesRefStr, &result, 0, map[string]struct{}{})

	return result
}

// validateFontResources recursively walks the Pages tree and checks font
// entries in each Page's /Resources dictionary for required fields.
func validateFontResources(f *os.File, pagesRef string, result *LevelResult, depth int, visited map[string]struct{}) {
	if depth > maxPageTreeDepth {
		return
	}
	if _, seen := visited[pagesRef]; seen {
		return
	}
	visited[pagesRef] = struct{}{}

	pagesObj, err := readObject(f, pagesRef)
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

	for _, kidRef := range ArrayToRefs(kidsArr) {
		refStr := RefToString(kidRef)
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
			validateFontResources(f, refStr, result, depth+1, visited)
			continue
		}

		if kidType != "Page" {
			continue
		}

		pageResources, _ := getPageResources(refStr, f)
		if pageResources == nil {
			continue
		}

		fontsDict, ok := DictGetDict(pageResources, "Font")
		if !ok {
			continue
		}

		for name, fontVal := range fontsDict {
			fontDictVal, ok := fontVal.(PDFDict)
			if !ok {
				fontRef, isRef := fontVal.(PDFRef)
				if !isRef {
					continue
				}
				fontObj, err := readObject(f, RefToString(fontRef))
				if err != nil {
					result.Warnings = append(result.Warnings, fmt.Sprintf("font %s: cannot read font object: %v", RefToString(fontRef), err))
					continue
				}
				fontDictVal, err = extractDictFromObject(fontObj)
				if err != nil {
					result.Warnings = append(result.Warnings, fmt.Sprintf("font %s: cannot extract font dict: %v", RefToString(fontRef), err))
					continue
				}
			}
			validateFontDict(fontDictVal, string(name), result)
		}
	}
}

// validateFontDict checks that a font dictionary has the required /Type, /Subtype,
// and /BaseFont fields. Missing /Type is a warning (since /Type may be inherited
// in some font types), missing /Subtype or /BaseFont are warnings.
func validateFontDict(fontDictVal PDFDict, name string, result *LevelResult) {
	if fontDictVal == nil {
		return
	}

	if typ, ok := DictGetName(fontDictVal, "Type"); !ok {
		result.Warnings = append(result.Warnings, fmt.Sprintf("font %s: /Type not found", name))
	} else if typ != "Font" {
		result.Errors = append(result.Errors, fmt.Sprintf("font %s: /Type is /%s, expected /Font", name, typ))
	}

	if _, ok := DictGetName(fontDictVal, "Subtype"); !ok {
		result.Warnings = append(result.Warnings, fmt.Sprintf("font %s: /Subtype not found", name))
	}

	if _, ok := DictGetName(fontDictVal, "BaseFont"); !ok {
		result.Warnings = append(result.Warnings, fmt.Sprintf("font %s: /BaseFont not found", name))
	}
}

// getPageResources returns the /Resources dict for a page, checking direct
// resources first then falling back to ancestor inheritance lookup.
func getPageResources(pageRefStr string, f *os.File) (PDFDict, bool) {
	pageObj, err := readObject(f, pageRefStr)
	if err != nil {
		return nil, false
	}
	pageDict, err := extractDictFromObject(pageObj)
	if err != nil {
		return nil, false
	}

	if resDict, ok := DictGetDict(pageDict, "Resources"); ok {
		return resDict, true
	}

	resRef, err := lookupResourcesFromAncestors(pageRefStr, f)
	if err != nil {
		return nil, false
	}

	resObj, err := readObject(f, RefToString(resRef))
	if err != nil {
		return nil, false
	}
	resDict, err := extractDictFromObject(resObj)
	if err != nil {
		return nil, false
	}
	return resDict, true
}

// LevelName returns human-readable name for a validation level.
func LevelName(level ValidationLevel) string {
	switch level {
	case LevelHeader:
		return "Header"
	case LevelXRef:
		return "XRef"
	case LevelTrailer:
		return "Trailer"
	case LevelCatalog:
		return "Catalog"
	case LevelPages:
		return "Pages"
	case LevelFonts:
		return "Fonts"
	default:
		return fmt.Sprintf("Level%d", level)
	}
}
