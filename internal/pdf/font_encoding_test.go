package pdf

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
	testfixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

// TestWinAnsiMapping tests WinAnsiEncoding byte mappings.
func TestWinAnsiMapping(t *testing.T) {
	tests := []struct {
		byteVal  byte
		expected rune
	}{
		// Windows extensions (0x80-0x9F)
		{0x80, '\u20AC'}, // Euro sign
		{0x85, '\u2026'}, // Ellipsis
		{0x91, '\u2018'}, // Left single quote
		{0x92, '\u2019'}, // Right single quote
		{0x93, '\u201C'}, // Left double quote
		{0x94, '\u201D'}, // Right double quote
		{0x95, '\u2022'}, // Bullet
		{0x96, '\u2013'}, // En dash
		{0x97, '\u2014'}, // Em dash
		{0x99, '\u2122'}, // Trademark

		// ISO-8859-1 compatible (0xA0-0xFF)
		{0xA0, '\u00A0'}, // Non-breaking space
		{0xA9, '\u00A9'}, // Copyright
		{0xC4, '\u00C4'}, // Ä
		{0xC5, '\u00C5'}, // Å
		{0xD6, '\u00D6'}, // Ö
		{0xDC, '\u00DC'}, // Ü
		{0xDF, '\u00DF'}, // ß (sharp s)
		{0xE4, '\u00E4'}, // ä
		{0xE9, '\u00E9'}, // é
		{0xFC, '\u00FC'}, // ü
		{0xF1, '\u00F1'}, // ñ
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("0x%02X", tt.byteVal), func(t *testing.T) {
			got, ok := winAnsiMapping[tt.byteVal]
			if !ok {
				t.Errorf("winAnsiMapping[0x%02X] not found", tt.byteVal)
				return
			}
			if got != tt.expected {
				t.Errorf("winAnsiMapping[0x%02X] = U+%04X, want U+%04X", tt.byteVal, got, tt.expected)
			}
		})
	}
}

// TestMacRomanMapping tests MacRomanEncoding byte mappings.
func TestMacRomanMapping(t *testing.T) {
	tests := []struct {
		byteVal  byte
		expected rune
	}{
		// Common MacRoman characters (different from WinAnsi)
		{0x80, '\u00C4'}, // Ä (WinAnsi has € at 0x80)
		{0x81, '\u00C5'}, // Å
		{0x87, '\u00E1'}, // á
		{0x8E, '\u00E9'}, // é (WinAnsi has Ž at 0x8E)
		{0x96, '\u00F1'}, // ñ
		{0x97, '\u00F3'}, // ó
		{0x9C, '\u00FA'}, // ú
		{0x9F, '\u00FC'}, // ü
		{0xE9, '\u00C8'}, // È (different from WinAnsi é!)

		// Special MacRoman characters
		{0xDB, '\u20AC'}, // Euro (in later MacRoman variants)
		{0xF0, '\u00D2'}, // Ò
		{0xF1, '\u00DA'}, // Ú
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("0x%02X", tt.byteVal), func(t *testing.T) {
			got, ok := macRomanMapping[tt.byteVal]
			if !ok {
				t.Errorf("macRomanMapping[0x%02X] not found", tt.byteVal)
				return
			}
			if got != tt.expected {
				t.Errorf("macRomanMapping[0x%02X] = U+%04X, want U+%04X", tt.byteVal, got, tt.expected)
			}
		})
	}
}

// TestWinAnsiVsMacRomanDifferences tests key differences between encodings.
func TestWinAnsiVsMacRomanDifferences(t *testing.T) {
	// 0x80: WinAnsi=€, MacRoman=Ä
	if winAnsiMapping[0x80] != '\u20AC' {
		t.Errorf("WinAnsi 0x80 should be Euro, got U+%04X", winAnsiMapping[0x80])
	}
	if macRomanMapping[0x80] != '\u00C4' {
		t.Errorf("MacRoman 0x80 should be Ä, got U+%04X", macRomanMapping[0x80])
	}

	// 0xE9: WinAnsi=é, MacRoman=È
	if winAnsiMapping[0xE9] != '\u00E9' {
		t.Errorf("WinAnsi 0xE9 should be é, got U+%04X", winAnsiMapping[0xE9])
	}
	if macRomanMapping[0xE9] != '\u00C8' {
		t.Errorf("MacRoman 0xE9 should be È, got U+%04X", macRomanMapping[0xE9])
	}
}

// TestApplyByteMapping tests the byte mapping function.
func TestApplyByteMapping(t *testing.T) {
	tests := []struct {
		name     string
		rawText  string
		mapping  map[byte]rune
		expected string
	}{
		{
			name:     "ASCII unchanged",
			rawText:  "Hello World",
			mapping:  winAnsiMapping,
			expected: "Hello World",
		},
		{
			name:     "WinAnsi high bytes",
			rawText:  string([]byte{0xC4, 0xE4, 0xFC}), // Ääü
			mapping:  winAnsiMapping,
			expected: "Ääü",
		},
		{
			name:     "MacRoman high bytes",
			rawText:  string([]byte{0x80, 0x87, 0x9F}), // Äáü
			mapping:  macRomanMapping,
			expected: "Äáü",
		},
		{
			name:     "mixed ASCII and high bytes",
			rawText:  "H" + string([]byte{0xE9}) + "llo", // Héllo
			mapping:  winAnsiMapping,
			expected: "Héllo",
		},
		{
			name:     "empty mapping",
			rawText:  "test",
			mapping:  map[byte]rune{},
			expected: "test",
		},
		{
			name:     "UTF-8 preserved",
			rawText:  "中文", // Already UTF-8, should be preserved
			mapping:  winAnsiMapping,
			expected: "中文",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyByteMapping(tt.rawText, tt.mapping)
			if got != tt.expected {
				t.Errorf("applyByteMapping(%q) = %q, want %q", tt.rawText, got, tt.expected)
			}
		})
	}
}

// TestApplyFontEncoding_BuiltinEncoding tests font encoding with built-in encodings.
func TestApplyFontEncoding_BuiltinEncoding(t *testing.T) {
	tests := []struct {
		name     string
		rawText  string
		font     FontInfo
		expected string
	}{
		{
			name:    "WinAnsiEncoding font",
			rawText: string([]byte{0xC4, 0xE4, 0xFC}), // Ääü
			font: FontInfo{
				Name:     "F1",
				Encoding: "WinAnsiEncoding",
				ToUnicode: nil,
			},
			expected: "Ääü",
		},
		{
			name:    "MacRomanEncoding font",
			rawText: string([]byte{0x80, 0x96, 0x9C}), // Äñú
			font: FontInfo{
				Name:     "F2",
				Encoding: "MacRomanEncoding",
				ToUnicode: nil,
			},
			expected: "Äñú",
		},
		{
			name:    "ToUnicode takes precedence over encoding",
			rawText: "AB",
			font: FontInfo{
				Name:     "F3",
				Encoding: "WinAnsiEncoding",
				ToUnicode: map[rune]string{
					'A': "第一章",
					'B': "第二章",
				},
			},
			expected: "第一章第二章",
		},
		{
			name:    "no encoding info - returns raw",
			rawText: "test",
			font: FontInfo{
				Name:      "F4",
				Encoding:  "",
				ToUnicode: nil,
			},
			expected: "test",
		},
		{
			name:    "MacExpertEncoding not supported - returns raw",
			rawText: "test",
			font: FontInfo{
				Name:      "F5",
				Encoding:  "MacExpertEncoding",
				ToUnicode: nil,
			},
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fonts := map[string]FontInfo{
				tt.font.Name: tt.font,
			}
			got := applyFontEncoding(tt.rawText, tt.font.Name, fonts)
			if got != tt.expected {
				t.Errorf("applyFontEncoding(%q, %q) = %q, want %q", tt.rawText, tt.font.Name, got, tt.expected)
			}
		})
	}
}

// TestApplyFontEncoding_Priority tests encoding priority (ToUnicode > Encoding > Raw).
func TestApplyFontEncoding_Priority(t *testing.T) {
	// Test priority: ToUnicode > WinAnsi > Raw
	font := FontInfo{
		Name:      "TestFont",
		Encoding:  "WinAnsiEncoding",
		ToUnicode: nil,
	}
	fonts := map[string]FontInfo{"TestFont": font}

	// Raw text with character 'A' (can be overridden by ToUnicode)
	rawText := "AB"

	// Should use WinAnsi encoding (ASCII passes through unchanged)
	got := applyFontEncoding(rawText, "TestFont", fonts)
	if got != "AB" {
		t.Errorf("WinAnsi encoding failed: got %q, want AB", got)
	}

	// Now add ToUnicode that overrides 'A' and 'B'
	font.ToUnicode = map[rune]string{'A': "[ALPHA]", 'B': "[BETA]"}
	fonts["TestFont"] = font

	got = applyFontEncoding(rawText, "TestFont", fonts)
	if got != "[ALPHA][BETA]" {
		t.Errorf("ToUnicode should take precedence: got %q, want [ALPHA][BETA]", got)
	}
}

// TestWinAnsiAccentedCharacters tests common accented characters in WinAnsi.
func TestWinAnsiAccentedCharacters(t *testing.T) {
	accents := map[byte]string{
		0xC0: "À", 0xC1: "Á", 0xC2: "Â", 0xC3: "Ã", 0xC4: "Ä", 0xC5: "Å",
		0xC8: "È", 0xC9: "É", 0xCA: "Ê", 0xCB: "Ë",
		0xCC: "Ì", 0xCD: "Í", 0xCE: "Î", 0xCF: "Ï",
		0xD2: "Ò", 0xD3: "Ó", 0xD4: "Ô", 0xD5: "Õ", 0xD6: "Ö",
		0xD9: "Ù", 0xDA: "Ú", 0xDB: "Û", 0xDC: "Ü",
		0xE0: "à", 0xE1: "á", 0xE2: "â", 0xE3: "ã", 0xE4: "ä", 0xE5: "å",
		0xE8: "è", 0xE9: "é", 0xEA: "ê", 0xEB: "ë",
		0xEC: "ì", 0xED: "í", 0xEE: "î", 0xEF: "ï",
		0xF2: "ò", 0xF3: "ó", 0xF4: "ô", 0xF5: "õ", 0xF6: "ö",
		0xF9: "ù", 0xFA: "ú", 0xFB: "û", 0xFC: "ü",
		0xFF: "ÿ",
	}

	for b, expected := range accents {
		rawText := string([]byte{b})
		got := applyByteMapping(rawText, winAnsiMapping)
		if got != expected {
			t.Errorf("WinAnsi 0x%02X: got %q, want %q", b, got, expected)
		}
	}
}

// TestEncodingWithRealPDF validates encoding with testdata PDFs.
func TestEncodingWithRealPDF(t *testing.T) {
	svc := NewService()

	// Use a standard PDF sample
	sample, ok := testfixtures.PDFSampleByKey("core-multipage")
	if !ok {
		t.Fatal("Sample 'core-multipage' not found")
	}

	doc, err := svc.Open(context.Background(), doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()})
	if err != nil {
		t.Fatalf("Failed to open PDF: %v", err)
	}
	defer doc.Close()

	result, err := svc.ExtractText(context.Background(), doc)
	if err != nil {
		t.Fatalf("ExtractText error: %v", err)
	}

	// Verify we got text (encoding should work)
	if result.Text == "" {
		t.Error("Expected non-empty text extraction")
	}

	// Check that the text contains expected content
	// The PDF should have decoded properly (not garbled)
	if strings.Contains(result.Text, "Sample") || strings.Contains(result.Text, "PDF") {
		t.Logf("Text extraction succeeded with proper encoding")
	}
}
