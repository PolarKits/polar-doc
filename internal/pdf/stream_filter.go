package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
)

// parseFilterNames extracts filter names from a stream dictionary string.
// It supports both single filters (/Filter /FlateDecode) and array filters
// (/Filter [/ASCII85Decode /FlateDecode]). Returns a slice of filter names
// without the leading '/'.
//
// Examples:
//   - "/Filter /FlateDecode" → ["FlateDecode"]
//   - "/Filter [/ASCII85Decode /FlateDecode]" → ["ASCII85Decode", "FlateDecode"]
//   - "" → [] (no filter)
func parseFilterNames(dictStr string) []string {
	// Find /Filter entry
	idx := strings.Index(dictStr, "/Filter")
	if idx < 0 {
		return nil
	}

	// Skip "/Filter" and whitespace
	start := idx + len("/Filter")
	for start < len(dictStr) && unicode.IsSpace(rune(dictStr[start])) {
		start++
	}
	if start >= len(dictStr) {
		return nil
	}

	// Check if it's an array
	if dictStr[start] == '[' {
		return parseFilterArray(dictStr[start:])
	}

	// Single filter
	name := parseName(dictStr[start:])
	if name != "" {
		return []string{name}
	}
	return nil
}

// parseFilterArray parses an array of filter names starting with '['.
func parseFilterArray(s string) []string {
	if len(s) == 0 || s[0] != '[' {
		return nil
	}

	var filters []string
	i := 1 // Skip '['
	depth := 1

	for i < len(s) && depth > 0 {
		// Skip whitespace
		for i < len(s) && unicode.IsSpace(rune(s[i])) {
			i++
		}
		if i >= len(s) {
			break
		}

		ch := s[i]
		if ch == ']' {
			depth--
			if depth == 0 {
				break
			}
			i++
		} else if ch == '[' {
			depth++
			i++
		} else if ch == '/' {
			// Found a name - parseName handles finding the end
			name := parseName(s[i:])
			if name != "" {
				filters = append(filters, name)
				// Move past this entire name entry (including the leading /)
				i += len(name) + 1
				continue
			}
			i++
		} else {
			i++
		}
	}

	return filters
}

// parseName extracts a PDF name starting with '/'.
// Returns the name without the leading '/'.
func parseName(s string) string {
	if len(s) == 0 || s[0] != '/' {
		return ""
	}

	i := 1
	for i < len(s) {
		ch := s[i]
		// PDF names end at whitespace or delimiter
		if unicode.IsSpace(rune(ch)) || ch == '[' || ch == ']' || ch == '/' || ch == '<' || ch == '>' {
			break
		}
		i++
	}

	if i > 1 {
		return s[1:i]
	}
	return ""
}

// decodeStream decodes stream data by applying filters in order.
// According to PDF spec, filters are applied in the order they appear in the array,
// so we decode in the same order (first filter in array is decoded first).
//
// Supported filters:
//   - FlateDecode: zlib decompression
//   - ASCIIHexDecode: hexadecimal decoding
//   - ASCII85Decode: Base85 (Adobe variant) decoding
//   - LZWDecode: LZW decompression
//   - RunLengthDecode: run-length encoding decoding
//   - CCITTFaxDecode / CCF: CCITT Group 3/4 fax compression (pass-through; not decoded)
//   - JBIG2Decode: JBIG2 bitonal image compression (pass-through; not decoded)
//   - DCTDecode: JPEG lossy image compression (pass-through; not decoded)
//   - JPXDecode: JPEG 2000 image compression (pass-through; not decoded)
//
// Returns an error if an unsupported filter is encountered.
func decodeStream(data []byte, filters []string) ([]byte, error) {
	if len(filters) == 0 {
		return data, nil
	}

	result := data
	for _, filter := range filters {
		var err error
		switch filter {
		case "FlateDecode":
			result, err = decodeFlate(result)
		case "ASCIIHexDecode":
			result, err = decodeASCIIHex(result)
		case "ASCII85Decode":
			result, err = decodeASCII85(result)
		case "LZWDecode":
			result, err = decodeLZW(result)
		case "RunLengthDecode":
			result, err = decodeRunLength(result)
		case "CCITTFaxDecode", "CCF":
			result, err = decodeCCITTFax(result)
		case "JBIG2Decode":
			result, err = decodeJBIG2(result)
		case "DCTDecode":
			result, err = decodeDCT(result)
		case "JPXDecode":
			result, err = decodeJPX(result)
		default:
			return nil, fmt.Errorf("unsupported filter: %s", filter)
		}
		if err != nil {
			return nil, fmt.Errorf("filter %s: %w", filter, err)
		}
	}

	return result, nil
}

// decodeFlate decompresses zlib (FlateDecode) encoded data.
func decodeFlate(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("zlib NewReader: %w", err)
	}
	defer r.Close()

	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("zlib ReadAll: %w", err)
	}
	return decompressed, nil
}

// decodeASCIIHex decodes ASCIIHexDecode encoded data.
// The format is hexadecimal digits with optional whitespace and
// an optional '>' terminator.
//
// Examples:
//   - "48656C6C6F>" → "Hello"
//   - "48 65 6C 6C 6F" → "Hello"
//   - "486" → "H0" (odd length padded with '0')
func decodeASCIIHex(data []byte) ([]byte, error) {
	// Remove all whitespace
	var hexStr strings.Builder
	for _, b := range data {
		if !unicode.IsSpace(rune(b)) && b != '>' {
			hexStr.WriteByte(b)
		}
	}

	s := hexStr.String()
	if len(s) == 0 {
		return []byte{}, nil
	}

	// Pad odd length with '0'
	if len(s)%2 == 1 {
		s += "0"
	}

	// Decode hex
	result := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		b, err := strconv.ParseUint(s[i:i+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex at position %d: %w", i, err)
		}
		result[i/2] = byte(b)
	}

	return result, nil
}

// decodeASCII85 decodes ASCII85 (Base85) encoded data using Adobe variant.
// The format uses 5 ASCII characters to represent 4 bytes.
// Special character 'z' represents 4 zero bytes.
// Optional <~ prefix and ~> suffix.
//
// Reference: Adobe PostScript Language Reference, Section 3.13.3
func decodeASCII85(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	s := string(data)

	// Strip optional <~ prefix and ~> suffix
	if strings.HasPrefix(s, "<~") {
		s = s[2:]
	}
	if strings.HasSuffix(s, "~>") {
		s = s[:len(s)-2]
	}

	// Remove whitespace
	var cleaned strings.Builder
	for _, ch := range s {
		if !unicode.IsSpace(ch) {
			cleaned.WriteRune(ch)
		}
	}
	s = cleaned.String()

	if len(s) == 0 {
		return []byte{}, nil
	}

	var result bytes.Buffer

	// Process in chunks of 5 characters
	i := 0
	for i < len(s) {
		// 'z' is a special shorthand for 4 zero bytes
		if s[i] == 'z' {
			result.Write([]byte{0, 0, 0, 0})
			i++
			continue
		}

		// Get up to 5 characters for this group
		end := i + 5
		if end > len(s) {
			end = len(s)
		}
		group := s[i:end]
		i = end

		// Decode this group
		var value uint32
		count := len(group)

		for _, ch := range group {
			// ASCII85 uses '!' (33) through 'u' (117)
			if ch < '!' || ch > 'u' {
				return nil, fmt.Errorf("invalid ASCII85 character: %c", ch)
			}
			value = value*85 + uint32(ch-'!')
		}

		// Calculate how many bytes to output
		// 5 chars → 4 bytes, 4 chars → 3 bytes, etc.
		bytesOut := count - 1
		if bytesOut < 1 {
			bytesOut = 1
		}

		// Write bytes (big-endian)
		out := make([]byte, 4)
		out[0] = byte(value >> 24)
		out[1] = byte(value >> 16)
		out[2] = byte(value >> 8)
		out[3] = byte(value)
		result.Write(out[:bytesOut])
	}

	return result.Bytes(), nil
}

// lzwEntry represents a single entry in the LZW code table.
// Each entry stores a prefix code and a suffix byte to reconstruct strings.
type lzwEntry struct {
	prefix uint16
	suffix byte
	length int
}

// decodeRunLength decodes RunLengthDecode encoded data per ISO 32000-2 §7.4.5.
//
// RunLengthDecode is a simple lossless compression algorithm where data consists
// of run-length encoded sequences. Each sequence begins with a length byte:
//   - 0-127: copy the next (n+1) bytes as-is
//   - 128: EOD (end of data) marker
//   - 129-255: copy the next byte (257-n) times
//
// Example encoded sequence: 0x05 0x48 0x45 0x4C 0x4C 0x4F (length=5, "HELLO")
// Example repeat sequence: 0xFB 0x41 (repeat 'A' 257-251=6 times)
func decodeRunLength(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	var result bytes.Buffer
	i := 0

	for i < len(data) {
		length := int(data[i])
		i++

		if length < 128 {
			// Literal run: next (length+1) bytes are copied as-is
			end := i + length + 1
			if end > len(data) {
				return nil, fmt.Errorf("run length decode: truncated literal run at index %d", i-1)
			}
			result.Write(data[i : i+length+1])
			i = end
		} else if length > 128 {
			// Repeat run: next byte is repeated (257-length) times
			if i >= len(data) {
				return nil, fmt.Errorf("run length decode: missing byte after repeat marker at index %d", i-1)
			}
			repeat := byte(257 - length)
			for j := 0; j < int(repeat); j++ {
				result.WriteByte(data[i])
			}
			i++
		} else {
			// length == 128: EOD marker
			break
		}
	}

	return result.Bytes(), nil
}

// decodeLZW decodes LZW compressed data.
// Uses standard LZW algorithm with initial code table size of 9 bits.
// The initial code table contains:
//   - Codes 0-255: single bytes
//   - Code 256: clear table marker
//   - Code 257: end of data marker
//
// Note: This implementation uses EarlyChange=1 behavior (default in PDF spec).
// When the code table fills, the code size increases immediately.
func decodeLZW(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return []byte{}, nil
	}

	// Bit reader for variable-length codes
	bits := &bitReader{data: data}

	// Initialize LZW state
	const (
		clearCode   = 256
		endCode     = 257
		firstCode   = 258
		maxCodeSize = 12
	)

	// Code table: maps code to byte sequence
	table := make([]lzwEntry, 1<<maxCodeSize)

	// Output buffer
	var output bytes.Buffer

	// Current code size and next available code
	codeSize := 9
	nextCode := firstCode

	// Previous code for adding new entries
	var prevCode uint16
	var prevSuffix byte
	firstChar := byte(0)

	// Initialize table with single-byte entries
	for i := 0; i < 256; i++ {
		table[i] = lzwEntry{prefix: 0, suffix: byte(i), length: 1}
	}

	// Clear table marker entry
	table[clearCode] = lzwEntry{prefix: 0, suffix: 0, length: 0}
	// End marker entry
	table[endCode] = lzwEntry{prefix: 0, suffix: 0, length: 0}

	// Read and process codes
	for {
		code, err := bits.readBits(codeSize)
		if err != nil {
			// End of stream - this is normal for some PDFs
			break
		}

		if code == clearCode {
			// Reset table
			codeSize = 9
			nextCode = firstCode
			prevCode = 0
			continue
		}

		if code == endCode {
			break
		}

		if code > uint16(nextCode) {
			// Invalid code
			return nil, fmt.Errorf("invalid LZW code: %d (max %d)", code, nextCode)
		}

		// Decode the code to bytes
		var decoded []byte
		if code < uint16(nextCode) {
			// Existing code
			decoded = getString(table, code)
		} else if code == uint16(nextCode) {
			// New code: previous code + first char of previous code
			if prevCode == 0 {
				return nil, fmt.Errorf("invalid LZW state: KwKw case with no previous code")
			}
			decoded = append(getString(table, prevCode), firstChar)
		} else {
			return nil, fmt.Errorf("invalid LZW code: %d >= nextCode %d", code, nextCode)
		}

		output.Write(decoded)
		firstChar = decoded[0]

		// Add new entry to table if we have a previous code
		if prevCode != 0 {
			table[nextCode] = lzwEntry{
				prefix: prevCode,
				suffix: firstChar,
				length: table[prevCode].length + 1,
			}
			nextCode++

			// Increase code size when table fills (EarlyChange=1)
			if nextCode >= (1<<codeSize) && codeSize < maxCodeSize {
				codeSize++
			}
		}

		prevCode = code
		prevSuffix = decoded[len(decoded)-1]
		_ = prevSuffix // May be used in future for optimization
	}

	return output.Bytes(), nil
}

// bitReader reads variable-length bits from a byte slice.
type bitReader struct {
	data   []byte
	pos    int // byte position
	bitPos int // bit position within current byte (0-7)
}

// readBits reads n bits from the stream and returns as uint16.
func (r *bitReader) readBits(n int) (uint16, error) {
	var result uint16
	for i := 0; i < n; i++ {
		if r.pos >= len(r.data) {
			return result, io.EOF
		}

		// Read bit (LSB first, per PDF LZW spec)
		bit := (r.data[r.pos] >> r.bitPos) & 1
		result |= uint16(bit) << i

		r.bitPos++
		if r.bitPos >= 8 {
			r.bitPos = 0
			r.pos++
		}
	}
	return result, nil
}

// getString reconstructs the byte sequence for a code from the LZW table.
func getString(table []lzwEntry, code uint16) []byte {
	if code < 256 {
		return []byte{byte(code)}
	}

	// Build string backwards (table entries reference prefix)
	var result []byte
	for code >= 256 {
		entry := table[code]
		result = append(result, entry.suffix)
		code = entry.prefix
	}
	result = append(result, byte(code))

	// Reverse
	for i, j := 0, len(result)-1; i < j; i, j = j, i {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// decodeCCITTFax is a stub for CCITT Group 3/4 fax decompression.
// Go's standard library does not include a CCITT decoder.
// The raw data is returned unchanged; callers should not expect decompressed output.
func decodeCCITTFax(data []byte) ([]byte, error) {
	return data, nil
}

// decodeJBIG2 is a stub for JBIG2 bitonal image decompression (ISO 32000-2 §7.4.7).
// Go's standard library does not include a JBIG2 decoder.
// The raw data is returned unchanged; callers should not expect decompressed output.
func decodeJBIG2(data []byte) ([]byte, error) {
	return data, nil
}

// decodeDCT is a stub for DCT (JPEG) image decompression (ISO 32000-2 §7.4.8).
// Go's standard library does not include a PDF DCT stream decoder.
// The raw data is returned unchanged; callers should not expect decompressed output.
func decodeDCT(data []byte) ([]byte, error) {
	return data, nil
}

// decodeJPX is a stub for JPEG 2000 image decompression (ISO 32000-2 §7.4.9).
// Go's standard library does not include a JPX stream decoder.
// The raw data is returned unchanged; callers should not expect decompressed output.
func decodeJPX(data []byte) ([]byte, error) {
	return data, nil
}
