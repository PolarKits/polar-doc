package pdf

import "strings"

// winAnsiMapping maps WinAnsiEncoding bytes 128-255 to Unicode runes.
// WinAnsiEncoding (Windows 1252) is the most common encoding in PDFs.
// Source: ISO 32000-1 Annex D.2, PDF 1.7 Reference
var winAnsiMapping = map[byte]rune{
	// 0x80-0x9F: Windows extensions (different from ISO-8859-1)
	0x80: '\u20AC', // Euro sign
	0x81: '\u0000', // Undefined (control)
	0x82: '\u201A', // Single low-9 quotation mark
	0x83: '\u0192', // Latin small letter f with hook
	0x84: '\u201E', // Double low-9 quotation mark
	0x85: '\u2026', // Horizontal ellipsis
	0x86: '\u2020', // Dagger
	0x87: '\u2021', // Double dagger
	0x88: '\u02C6', // Modifier letter circumflex accent
	0x89: '\u2030', // Per mille sign
	0x8A: '\u0160', // Latin capital letter S with caron
	0x8B: '\u2039', // Single left-pointing angle quotation mark
	0x8C: '\u0152', // Latin capital ligature OE
	0x8D: '\u0000', // Undefined (control)
	0x8E: '\u017D', // Latin capital letter Z with caron
	0x8F: '\u0000', // Undefined (control)
	0x90: '\u0000', // Undefined (control)
	0x91: '\u2018', // Left single quotation mark
	0x92: '\u2019', // Right single quotation mark
	0x93: '\u201C', // Left double quotation mark
	0x94: '\u201D', // Right double quotation mark
	0x95: '\u2022', // Bullet
	0x96: '\u2013', // En dash
	0x97: '\u2014', // Em dash
	0x98: '\u02DC', // Small tilde
	0x99: '\u2122', // Trade mark sign
	0x9A: '\u0161', // Latin small letter s with caron
	0x9B: '\u203A', // Single right-pointing angle quotation mark
	0x9C: '\u0153', // Latin small ligature oe
	0x9D: '\u0000', // Undefined (control)
	0x9E: '\u017E', // Latin small letter z with caron
	0x9F: '\u0178', // Latin capital letter Y with diaeresis

	// 0xA0-0xFF: ISO-8859-1 compatible
	0xA0: '\u00A0', // Non-breaking space
	0xA1: '\u00A1', // Inverted exclamation mark
	0xA2: '\u00A2', // Cent sign
	0xA3: '\u00A3', // Pound sign
	0xA4: '\u00A4', // Currency sign
	0xA5: '\u00A5', // Yen sign
	0xA6: '\u00A6', // Broken bar
	0xA7: '\u00A7', // Section sign
	0xA8: '\u00A8', // Diaeresis
	0xA9: '\u00A9', // Copyright sign
	0xAA: '\u00AA', // Feminine ordinal indicator
	0xAB: '\u00AB', // Left-pointing double angle quotation mark
	0xAC: '\u00AC', // Not sign
	0xAD: '\u00AD', // Soft hyphen
	0xAE: '\u00AE', // Registered sign
	0xAF: '\u00AF', // Macron
	0xB0: '\u00B0', // Degree sign
	0xB1: '\u00B1', // Plus-minus sign
	0xB2: '\u00B2', // Superscript two
	0xB3: '\u00B3', // Superscript three
	0xB4: '\u00B4', // Acute accent
	0xB5: '\u00B5', // Micro sign
	0xB6: '\u00B6', // Pilcrow sign
	0xB7: '\u00B7', // Middle dot
	0xB8: '\u00B8', // Cedilla
	0xB9: '\u00B9', // Superscript one
	0xBA: '\u00BA', // Masculine ordinal indicator
	0xBB: '\u00BB', // Right-pointing double angle quotation mark
	0xBC: '\u00BC', // Vulgar fraction one quarter
	0xBD: '\u00BD', // Vulgar fraction one half
	0xBE: '\u00BE', // Vulgar fraction three quarters
	0xBF: '\u00BF', // Inverted question mark
	0xC0: '\u00C0', // Latin capital letter A with grave
	0xC1: '\u00C1', // Latin capital letter A with acute
	0xC2: '\u00C2', // Latin capital letter A with circumflex
	0xC3: '\u00C3', // Latin capital letter A with tilde
	0xC4: '\u00C4', // Latin capital letter A with diaeresis
	0xC5: '\u00C5', // Latin capital letter A with ring above
	0xC6: '\u00C6', // Latin capital ligature AE
	0xC7: '\u00C7', // Latin capital letter C with cedilla
	0xC8: '\u00C8', // Latin capital letter E with grave
	0xC9: '\u00C9', // Latin capital letter E with acute
	0xCA: '\u00CA', // Latin capital letter E with circumflex
	0xCB: '\u00CB', // Latin capital letter E with diaeresis
	0xCC: '\u00CC', // Latin capital letter I with grave
	0xCD: '\u00CD', // Latin capital letter I with acute
	0xCE: '\u00CE', // Latin capital letter I with circumflex
	0xCF: '\u00CF', // Latin capital letter I with diaeresis
	0xD0: '\u00D0', // Latin capital letter Eth
	0xD1: '\u00D1', // Latin capital letter N with tilde
	0xD2: '\u00D2', // Latin capital letter O with grave
	0xD3: '\u00D3', // Latin capital letter O with acute
	0xD4: '\u00D4', // Latin capital letter O with circumflex
	0xD5: '\u00D5', // Latin capital letter O with tilde
	0xD6: '\u00D6', // Latin capital letter O with diaeresis
	0xD7: '\u00D7', // Multiplication sign
	0xD8: '\u00D8', // Latin capital letter O with stroke
	0xD9: '\u00D9', // Latin capital letter U with grave
	0xDA: '\u00DA', // Latin capital letter U with acute
	0xDB: '\u00DB', // Latin capital letter U with circumflex
	0xDC: '\u00DC', // Latin capital letter U with diaeresis
	0xDD: '\u00DD', // Latin capital letter Y with acute
	0xDE: '\u00DE', // Latin capital letter Thorn
	0xDF: '\u00DF', // Latin small letter sharp s
	0xE0: '\u00E0', // Latin small letter a with grave
	0xE1: '\u00E1', // Latin small letter a with acute
	0xE2: '\u00E2', // Latin small letter a with circumflex
	0xE3: '\u00E3', // Latin small letter a with tilde
	0xE4: '\u00E4', // Latin small letter a with diaeresis
	0xE5: '\u00E5', // Latin small letter a with ring above
	0xE6: '\u00E6', // Latin small ligature ae
	0xE7: '\u00E7', // Latin small letter c with cedilla
	0xE8: '\u00E8', // Latin small letter e with grave
	0xE9: '\u00E9', // Latin small letter e with acute
	0xEA: '\u00EA', // Latin small letter e with circumflex
	0xEB: '\u00EB', // Latin small letter e with diaeresis
	0xEC: '\u00EC', // Latin small letter i with grave
	0xED: '\u00ED', // Latin small letter i with acute
	0xEE: '\u00EE', // Latin small letter i with circumflex
	0xEF: '\u00EF', // Latin small letter i with diaeresis
	0xF0: '\u00F0', // Latin small letter eth
	0xF1: '\u00F1', // Latin small letter n with tilde
	0xF2: '\u00F2', // Latin small letter o with grave
	0xF3: '\u00F3', // Latin small letter o with acute
	0xF4: '\u00F4', // Latin small letter o with circumflex
	0xF5: '\u00F5', // Latin small letter o with tilde
	0xF6: '\u00F6', // Latin small letter o with diaeresis
	0xF7: '\u00F7', // Division sign
	0xF8: '\u00F8', // Latin small letter o with stroke
	0xF9: '\u00F9', // Latin small letter u with grave
	0xFA: '\u00FA', // Latin small letter u with acute
	0xFB: '\u00FB', // Latin small letter u with circumflex
	0xFC: '\u00FC', // Latin small letter u with diaeresis
	0xFD: '\u00FD', // Latin small letter y with acute
	0xFE: '\u00FE', // Latin small letter thorn
	0xFF: '\u00FF', // Latin small letter y with diaeresis
}

// macRomanMapping maps MacRomanEncoding bytes 128-255 to Unicode runes.
// MacRomanEncoding is used in PDFs from macOS systems.
// Source: ISO 32000-1 Annex D.2, PDF 1.7 Reference
var macRomanMapping = map[byte]rune{
	// MacRoman encoding table (128-255)
	0x80: '\u00C4', // Latin capital letter A with diaeresis (Ä)
	0x81: '\u00C5', // Latin capital letter A with ring above (Å)
	0x82: '\u00C7', // Latin capital letter C with cedilla (Ç)
	0x83: '\u00C9', // Latin capital letter E with acute (É)
	0x84: '\u00D1', // Latin capital letter N with tilde (Ñ)
	0x85: '\u00D6', // Latin capital letter O with diaeresis (Ö)
	0x86: '\u00DC', // Latin capital letter U with diaeresis (Ü)
	0x87: '\u00E1', // Latin small letter a with acute (á)
	0x88: '\u00E0', // Latin small letter a with grave (à)
	0x89: '\u00E2', // Latin small letter a with circumflex (â)
	0x8A: '\u00E4', // Latin small letter a with diaeresis (ä)
	0x8B: '\u00E3', // Latin small letter a with tilde (ã)
	0x8C: '\u00E5', // Latin small letter a with ring above (å)
	0x8D: '\u00E7', // Latin small letter c with cedilla (ç)
	0x8E: '\u00E9', // Latin small letter e with acute (é)
	0x8F: '\u00E8', // Latin small letter e with grave (è)
	0x90: '\u00EA', // Latin small letter e with circumflex (ê)
	0x91: '\u00EB', // Latin small letter e with diaeresis (ë)
	0x92: '\u00ED', // Latin small letter i with acute (í)
	0x93: '\u00EC', // Latin small letter i with grave (ì)
	0x94: '\u00EE', // Latin small letter i with circumflex (î)
	0x95: '\u00EF', // Latin small letter i with diaeresis (ï)
	0x96: '\u00F1', // Latin small letter n with tilde (ñ)
	0x97: '\u00F3', // Latin small letter o with acute (ó)
	0x98: '\u00F2', // Latin small letter o with grave (ò)
	0x99: '\u00F4', // Latin small letter o with circumflex (ô)
	0x9A: '\u00F6', // Latin small letter o with diaeresis (ö)
	0x9B: '\u00F5', // Latin small letter o with tilde (õ)
	0x9C: '\u00FA', // Latin small letter u with acute (ú)
	0x9D: '\u00F9', // Latin small letter u with grave (ù)
	0x9E: '\u00FB', // Latin small letter u with circumflex (û)
	0x9F: '\u00FC', // Latin small letter u with diaeresis (ü)
	0xA0: '\u2020', // Dagger (†)
	0xA1: '\u00B0', // Degree sign (°)
	0xA2: '\u00A2', // Cent sign (¢)
	0xA3: '\u00A3', // Pound sign (£)
	0xA4: '\u00A7', // Section sign (§)
	0xA5: '\u2022', // Bullet (•)
	0xA6: '\u00B6', // Pilcrow sign (¶)
	0xA7: '\u00DF', // Latin small letter sharp s (ß)
	0xA8: '\u00AE', // Registered sign (®)
	0xA9: '\u00A9', // Copyright sign (©)
	0xAA: '\u2122', // Trade mark sign (™)
	0xAB: '\u00B4', // Acute accent (´)
	0xAC: '\u00A8', // Diaeresis (¨)
	0xAD: '\u2260', // Not equal to (≠)
	0xAE: '\u00C6', // Latin capital ligature AE (Æ)
	0xAF: '\u00D8', // Latin capital letter O with stroke (Ø)
	0xB0: '\u221E', // Infinity (∞)
	0xB1: '\u00B1', // Plus-minus sign (±)
	0xB2: '\u2264', // Less-than or equal to (≤)
	0xB3: '\u2265', // Greater-than or equal to (≥)
	0xB4: '\u00A5', // Yen sign (¥)
	0xB5: '\u00B5', // Micro sign (µ)
	0xB6: '\u2202', // Partial differential (∂)
	0xB7: '\u2211', // N-ary summation (∑)
	0xB8: '\u220F', // N-ary product (∏)
	0xB9: '\u03C0', // Greek small letter pi (π)
	0xBA: '\u222B', // Integral (∫)
	0xBB: '\u00AA', // Feminine ordinal indicator (ª)
	0xBC: '\u00BA', // Masculine ordinal indicator (º)
	0xBD: '\u03A9', // Greek capital letter omega (Ω)
	0xBE: '\u00E6', // Latin small ligature ae (æ)
	0xBF: '\u00F8', // Latin small letter o with stroke (ø)
	0xC0: '\u00BF', // Inverted question mark (¿)
	0xC1: '\u00A1', // Inverted exclamation mark (¡)
	0xC2: '\u00AC', // Not sign (¬)
	0xC3: '\u221A', // Square root (√)
	0xC4: '\u0192', // Latin small letter f with hook (ƒ)
	0xC5: '\u2248', // Almost equal to (≈)
	0xC6: '\u2206', // Increment (∆)
	0xC7: '\u00AB', // Left-pointing double angle quotation mark («)
	0xC8: '\u00BB', // Right-pointing double angle quotation mark (»)
	0xC9: '\u2026', // Horizontal ellipsis (…)
	0xCA: '\u00A0', // Non-breaking space
	0xCB: '\u00C0', // Latin capital letter A with grave (À)
	0xCC: '\u00C3', // Latin capital letter A with tilde (Ã)
	0xCD: '\u00D5', // Latin capital letter O with tilde (Õ)
	0xCE: '\u0152', // Latin capital ligature OE (Œ)
	0xCF: '\u0153', // Latin small ligature oe (œ)
	0xD0: '\u2013', // En dash (–)
	0xD1: '\u2014', // Em dash (—)
	0xD2: '\u201C', // Left double quotation mark (" )
	0xD3: '\u201D', // Right double quotation mark (")
	0xD4: '\u2018', // Left single quotation mark (' )
	0xD5: '\u2019', // Right single quotation mark (')
	0xD6: '\u00F7', // Division sign (÷)
	0xD7: '\u25CA', // Lozenge (◊)
	0xD8: '\u00FF', // Latin small letter y with diaeresis (ÿ)
	0xD9: '\u0178', // Latin capital letter Y with diaeresis (Ÿ)
	0xDA: '\u2044', // Fraction slash (⁄)
	0xDB: '\u20AC', // Euro sign (€)
	0xDC: '\u2039', // Single left-pointing angle quotation mark (‹)
	0xDD: '\u203A', // Single right-pointing angle quotation mark (›)
	0xDE: '\uFB01', // Latin small ligature fi (ﬁ)
	0xDF: '\uFB02', // Latin small ligature fl (ﬂ)
	0xE0: '\u2021', // Double dagger (‡)
	0xE1: '\u00B7', // Middle dot (·)
	0xE2: '\u201A', // Single low-9 quotation mark (‚)
	0xE3: '\u201E', // Double low-9 quotation mark („)
	0xE4: '\u2030', // Per mille sign (‰)
	0xE5: '\u00C2', // Latin capital letter A with circumflex (Â)
	0xE6: '\u00CA', // Latin capital letter E with circumflex (Ê)
	0xE7: '\u00C1', // Latin capital letter A with acute (Á)
	0xE8: '\u00CB', // Latin capital letter E with diaeresis (Ë)
	0xE9: '\u00C8', // Latin capital letter E with grave (È)
	0xEA: '\u00CD', // Latin capital letter I with acute (Í)
	0xEB: '\u00CE', // Latin capital letter I with circumflex (Î)
	0xEC: '\u00CF', // Latin capital letter I with diaeresis (Ï)
	0xED: '\u00CC', // Latin capital letter I with grave (Ì)
	0xEE: '\u00D3', // Latin capital letter O with acute (Ó)
	0xEF: '\u00D4', // Latin capital letter O with circumflex (Ô)
	0xF0: '\u00D2', // Latin capital letter O with grave (Ò)
	0xF1: '\u00DA', // Latin capital letter U with acute (Ú)
	0xF2: '\u00DB', // Latin capital letter U with circumflex (Û)
	0xF3: '\u00D9', // Latin capital letter U with grave (Ù)
	0xF4: '\u0131', // Latin small letter dotless i (ı)
	0xF5: '\u02C6', // Modifier letter circumflex accent (ˆ)
	0xF6: '\u02DC', // Small tilde (˜)
	0xF7: '\u00AF', // Macron (¯)
	0xF8: '\u02D8', // Breve (˘)
	0xF9: '\u02D9', // Dot above (˙)
	0xFA: '\u02DA', // Ring above (˚)
	0xFB: '\u00B8', // Cedilla (¸)
	0xFC: '\u02DB', // Ogonek (˛)
	0xFD: '\u02C7', // Caron (ˇ)
	0xFE: '\u201E', // Double low-9 quotation mark („)
	0xFF: '\u00A0', // Non-breaking space (duplicate of 0xCA in some variants)
}

// applyByteMapping applies a byte-to-rune mapping to raw text bytes.
// Bytes in the 0-127 ASCII range are preserved unchanged.
// Bytes in the 128-255 range are mapped using the provided mapping table.
// Multi-byte UTF-8 sequences (detected by high bit pattern) are preserved.
func applyByteMapping(rawText string, mapping map[byte]rune) string {
	var result strings.Builder

	for i := 0; i < len(rawText); i++ {
		b := rawText[i]

		// Check if this is a multi-byte UTF-8 sequence (0xC0-0xFF start byte)
		if b >= 0xC0 && b <= 0xFF {
			// Check if it's a valid UTF-8 start byte followed by continuation bytes
			// UTF-8: 110xxxxx (0xC0-0xDF) = 2 bytes, 1110xxxx (0xE0-0xEF) = 3 bytes
			// 11110xxx (0xF0-0xF7) = 4 bytes
			utf8Len := 0
			if b >= 0xC0 && b <= 0xDF {
				utf8Len = 2
			} else if b >= 0xE0 && b <= 0xEF {
				utf8Len = 3
			} else if b >= 0xF0 && b <= 0xF7 {
				utf8Len = 4
			}

			// Check if we have enough continuation bytes
			if utf8Len > 1 && i+utf8Len <= len(rawText) {
				validUTF8 := true
				for j := 1; j < utf8Len; j++ {
					nextByte := rawText[i+j]
					if nextByte < 0x80 || nextByte > 0xBF {
						validUTF8 = false
						break
					}
				}
				if validUTF8 {
					// Preserve the multi-byte UTF-8 sequence
					result.WriteString(rawText[i : i+utf8Len])
					i += utf8Len - 1
					continue
				}
			}
		}

		// Apply mapping for high bytes (128-255)
		if b >= 128 {
			if mappedRune, ok := mapping[b]; ok && mappedRune != 0 {
				result.WriteRune(mappedRune)
				continue
			}
		}

		// Keep ASCII and unmapped bytes as-is
		result.WriteByte(b)
	}

	return result.String()
}
