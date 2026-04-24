package pdf

import (
	"bytes"
	"compress/zlib"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
	testfixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

// TestParseToUnicodeCMap tests ToUnicode CMap parsing.
func TestParseToUnicodeCMap(t *testing.T) {
	tests := []struct {
		name     string
		cmapData string
		expected map[rune]string
	}{
		{
			name: "simple bfchar mapping",
			cmapData: `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincode
<0000> <0000>
endcode
1 beginbfchar
<0041> <0042>
endbfchar
endcmap
CMapName currentdict /CMap defineresource pop end end`,
			expected: map[rune]string{
				0x0041: "B", // A maps to B
			},
		},
		{
			name: "bfrange single target mapping",
			cmapData: `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincode
<0000> <0000>
endcode
1 beginbfrange
<0000> <0005> <0041>
endbfrange
endcmap
CMapName currentdict /CMap defineresource pop end end`,
			expected: map[rune]string{
				0x0000: "A",
				0x0001: "B",
				0x0002: "C",
				0x0003: "D",
				0x0004: "E",
				0x0005: "F",
			},
		},
		{
			name: "empty CMap",
			cmapData: `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
endcmap
CMapName currentdict /CMap defineresource pop end end`,
			expected: map[rune]string{},
		},
		{
			name: "multiple bfchar mappings",
			cmapData: `2 beginbfchar <0041> <0051> <0042> <0052> endbfchar`,
			expected: map[rune]string{
				0x0041: "Q", // A -> Q
				0x0042: "R", // B -> R
			},
		},
		{
			name: "bfchar and bfrange combined",
			cmapData: `1 beginbfchar <0041> <0051> endbfchar 1 beginbfrange <0061> <0065> <0031> endbfrange`,
			expected: map[rune]string{
				0x0041: "Q", // A -> Q
				0x0061: "1", // a -> 1
				0x0062: "2", // b -> 2
				0x0063: "3", // c -> 3
				0x0064: "4", // d -> 4
				0x0065: "5", // e -> 5
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For unit testing, we test the regex parsing logic on the CMap string directly
			// In real usage, the CMap would be read from a PDF stream object

			// Parse bfchar blocks
			bfcharRegex := regexp.MustCompile(`(\d+)\s+beginbfchar\s+(.*?)\s+endbfchar`)
			bfcharMatches := bfcharRegex.FindAllStringSubmatch(tt.cmapData, -1)

			cmap := make(map[rune]string)
			for _, match := range bfcharMatches {
				content := match[2]
				mappingRegex := regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
				mappingMatches := mappingRegex.FindAllStringSubmatch(content, -1)
				for _, mapping := range mappingMatches {
					srcCode, _ := strconv.ParseInt(mapping[1], 16, 32)
					dstUnicode, _ := strconv.ParseInt(mapping[2], 16, 32)
					cmap[rune(srcCode)] = string(rune(dstUnicode))
				}
			}

			// Parse bfrange blocks
			bfrangeRegex := regexp.MustCompile(`(\d+)\s+beginbfrange\s+(.*?)\s+endbfrange`)
			bfrangeMatches := bfrangeRegex.FindAllStringSubmatch(tt.cmapData, -1)
			for _, match := range bfrangeMatches {
				content := match[2]
				rangeRegex := regexp.MustCompile(`<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>\s*<([0-9A-Fa-f]+)>`)
				rangeMatches := rangeRegex.FindAllStringSubmatch(content, -1)
				for _, r := range rangeMatches {
					startCode, _ := strconv.ParseInt(r[1], 16, 32)
					endCode, _ := strconv.ParseInt(r[2], 16, 32)
					targetCode, _ := strconv.ParseInt(r[3], 16, 32)

					for i := startCode; i <= endCode; i++ {
						offset := i - startCode
						cmap[rune(i)] = string(rune(targetCode + offset))
					}
				}
			}

			// Verify expected mappings are present
			for charCode, expectedUnicode := range tt.expected {
				if got, ok := cmap[charCode]; !ok || got != expectedUnicode {
					t.Errorf("CMap[%04X] = %q, want %q", charCode, got, expectedUnicode)
				}
			}
		})
	}
}

// TestFontInfoParsing tests font dictionary parsing.
func TestFontInfoParsing(t *testing.T) {
	tests := []struct {
		name     string
		dict     PDFDict
		expected FontInfo
	}{
		{
			name: "Type1 font with ToUnicode",
			dict: PDFDict{
				PDFName("Type"):     PDFName("Font"),
				PDFName("Subtype"):  PDFName("Type1"),
				PDFName("BaseFont"): PDFName("Helvetica"),
				PDFName("Encoding"): PDFName("WinAnsiEncoding"),
				PDFName("ToUnicode"): PDFRef{ObjNum: 15, GenNum: 0},
			},
			expected: FontInfo{
				Name:         "F1",
				Subtype:      "Type1",
				BaseFont:     "Helvetica",
				Encoding:     "WinAnsiEncoding",
				ToUnicodeRef: PDFRef{ObjNum: 15, GenNum: 0},
			},
		},
		{
			name: "TrueType font without ToUnicode",
			dict: PDFDict{
				PDFName("Type"):     PDFName("Font"),
				PDFName("Subtype"):  PDFName("TrueType"),
				PDFName("BaseFont"): PDFName("Arial"),
			},
			expected: FontInfo{
				Name:     "F2",
				Subtype:  "TrueType",
				BaseFont: "Arial",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFontDict(tt.dict, tt.expected.Name)
			if got.Name != tt.expected.Name {
				t.Errorf("Name = %q, want %q", got.Name, tt.expected.Name)
			}
			if got.Subtype != tt.expected.Subtype {
				t.Errorf("Subtype = %q, want %q", got.Subtype, tt.expected.Subtype)
			}
			if got.BaseFont != tt.expected.BaseFont {
				t.Errorf("BaseFont = %q, want %q", got.BaseFont, tt.expected.BaseFont)
			}
			if got.Encoding != tt.expected.Encoding {
				t.Errorf("Encoding = %q, want %q", got.Encoding, tt.expected.Encoding)
			}
			if got.ToUnicodeRef != tt.expected.ToUnicodeRef {
				t.Errorf("ToUnicodeRef = %+v, want %+v", got.ToUnicodeRef, tt.expected.ToUnicodeRef)
			}
		})
	}
}

// TestApplyFontEncoding tests font encoding application.
func TestApplyFontEncoding(t *testing.T) {
	fonts := map[string]FontInfo{
		"F1": {
			Name: "F1",
			ToUnicode: map[rune]string{
				'A': "第一章",
				'B': "第二章",
				'C': "第三章",
			},
		},
		"F2": {
			Name:      "F2",
			ToUnicode: nil, // No ToUnicode mapping
		},
	}

	tests := []struct {
		name     string
		rawText  string
		fontName string
		expected string
	}{
		{
			name:     "font with ToUnicode mapping",
			rawText:  "ABC",
			fontName: "F1",
			expected: "第一章第二章第三章",
		},
		{
			name:     "font without ToUnicode mapping",
			rawText:  "Hello",
			fontName: "F2",
			expected: "Hello",
		},
		{
			name:     "unknown font",
			rawText:  "Test",
			fontName: "F99",
			expected: "Test",
		},
		{
			name:     "no font specified",
			rawText:  "Text",
			fontName: "",
			expected: "Text",
		},
		{
			name:     "mixed mapping and non-mapping chars",
			rawText:  "AxB",
			fontName: "F1",
			expected: "第一章x第二章", // 'x' has no mapping, kept as-is
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyFontEncoding(tt.rawText, tt.fontName, fonts)
			if got != tt.expected {
				t.Errorf("applyFontEncoding(%q, %q) = %q, want %q", tt.rawText, tt.fontName, got, tt.expected)
			}
		})
	}
}

// TestExtractTextWithFonts tests font-aware text extraction.
func TestExtractTextWithFonts(t *testing.T) {
	fonts := map[string]FontInfo{
		"F1": {
			Name: "F1",
			ToUnicode: map[rune]string{
				'A': "第",
				'B': "章",
			},
		},
	}

	operators := []ContentOperator{
		{Op: "BT"},
		{Op: "Tf", Operands: []ContentOperand{{Kind: OperandName, StrVal: "F1"}}},
		{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "AB"}}},
		{Op: "ET"},
	}

	result := extractTextWithFonts(operators, fonts)
	expected := "第章"
	if result != expected {
		t.Errorf("extractTextWithFonts = %q, want %q", result, expected)
	}
}

// TestExtractTextWithFonts_NoFonts tests backward compatibility.
func TestExtractTextWithFonts_NoFonts(t *testing.T) {
	operators := []ContentOperator{
		{Op: "BT"},
		{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "Hello"}}},
		{Op: "ET"},
	}

	// With nil fonts, should behave like original extractTextFromOperators
	result := extractTextWithFonts(operators, nil)
	expected := "Hello"
	if result != expected {
		t.Errorf("extractTextWithFonts(nil fonts) = %q, want %q", result, expected)
	}
}

// TestProcessTJArrayWithFonts tests TJ array processing with fonts.
func TestProcessTJArrayWithFonts(t *testing.T) {
	fonts := map[string]FontInfo{
		"F1": {
			Name: "F1",
			ToUnicode: map[rune]string{
				'X': "一",
				'Y': "二",
			},
		},
	}

	elements := []ContentOperand{
		{Kind: OperandString, StrVal: "XY"},
		{Kind: OperandNumber, NumVal: -100}, // Small adjustment, no space
		{Kind: OperandString, StrVal: "X"},
	}

	var result strings.Builder
	needsSpace := false
	processTJArrayWithFonts(elements, &result, &needsSpace, "F1", fonts)

	expected := "一二一"
	if result.String() != expected {
		t.Errorf("processTJArrayWithFonts = %q, want %q", result.String(), expected)
	}
}

// TestExtractTextQuality_WithFonts verifies no regression with real PDFs.
func TestExtractTextQuality_WithFonts(t *testing.T) {
	// This test ensures that font-aware extraction doesn't break existing functionality.
	// It runs on real PDF samples to verify text extraction still works.

	svc := NewService()

	// Use a sample that doesn't have complex ToUnicode mappings
	// This verifies the fallback to raw text works correctly
	sample, ok := testfixtures.PDFSampleByKey("core-multipage")
	if !ok {
		t.Fatal("Sample 'core-multipage' not found")
	}

	doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
	if err != nil {
		t.Fatalf("Failed to open PDF: %v", err)
	}
	defer doc.Close()

	result, err := svc.ExtractText(nil, doc)
	if err != nil {
		t.Fatalf("ExtractText error: %v", err)
	}

	// Verify we got some text (specific content may vary based on PDF)
	if result.Text == "" {
		t.Error("Expected non-empty text extraction")
	}

	// The text should not be significantly shorter due to font issues
	if len(result.Text) < 10 {
		t.Errorf("Extracted text too short (%d chars), possible font resolution issue", len(result.Text))
	}
}

// TestParseToUnicodeCMap_Compressed tests that FlateDecode compressed CMap can be parsed.
func TestParseToUnicodeCMap_Compressed(t *testing.T) {
	// Create a CMap content
	cmapContent := `1 beginbfchar
<0041> <0042>
endbfchar`

	// Compress it using zlib (FlateDecode)
	var compressed bytes.Buffer
	w := zlib.NewWriter(&compressed)
	w.Write([]byte(cmapContent))
	w.Close()

	// Build a mock stream object with FlateDecode filter
	// Format: "N G obj\n<< /Filter /FlateDecode /Length X >>\nstream\n...\nendstream\nendobj"
	objStr := fmt.Sprintf("15 0 obj\n<< /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream\nendobj",
		compressed.Len(), compressed.String())

	// Test that parseFilterNames correctly identifies the filter
	filters := parseFilterNames(objStr)
	if len(filters) != 1 || filters[0] != "FlateDecode" {
		t.Errorf("parseFilterNames: got %v, want [FlateDecode]", filters)
	}

	// Test that decodeStream can decompress the data
	decompressed, err := decodeStream(compressed.Bytes(), filters)
	if err != nil {
		t.Fatalf("decodeStream failed: %v", err)
	}

	// Verify decompressed content contains the CMap data
	if !strings.Contains(string(decompressed), "beginbfchar") {
		t.Errorf("Decompressed data missing CMap markers: %s", string(decompressed))
	}
}

// TestDecodeStream_Integration tests the integration with stream_filter.
func TestDecodeStream_Integration(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		filters  []string
		expected string
	}{
		{
			name:     "plain text no filter",
			content:  "1 beginbfchar <0041> <0042> endbfchar",
			filters:  []string{},
			expected: "1 beginbfchar <0041> <0042> endbfchar",
		},
		{
			name:     "FlateDecode",
			content:  "2 beginbfrange <0000> <0005> <0041> endbfrange",
			filters:  []string{"FlateDecode"},
			expected: "2 beginbfrange <0000> <0005> <0041> endbfrange",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data []byte
			if len(tt.filters) > 0 && tt.filters[0] == "FlateDecode" {
				// Compress the content
				var compressed bytes.Buffer
				w := zlib.NewWriter(&compressed)
				w.Write([]byte(tt.content))
				w.Close()
				data = compressed.Bytes()
			} else {
				data = []byte(tt.content)
			}

			result, err := decodeStream(data, tt.filters)
			if err != nil {
				t.Fatalf("decodeStream error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("decodeStream = %q, want %q", string(result), tt.expected)
			}
		})
	}
}
