package pdf

import (
	"fmt"
	"strconv"
	"strings"
)

// OperandKind identifies the type of a PDF content stream operand.
type OperandKind int

const (
	// OperandString represents a literal or hex string operand (e.g., "(Hello)" or "<48656C6C6F>").
	OperandString OperandKind = iota
	// OperandNumber represents a numeric operand (integer or real).
	OperandNumber
	// OperandArray represents an array operand ([...]).
	OperandArray
	// OperandName represents a name operand (e.g., "/Font").
	OperandName
	// OperandRef represents an indirect object reference (e.g., "1 0 R").
	OperandRef
)

// String returns a human-readable name for the operand kind.
func (k OperandKind) String() string {
	switch k {
	case OperandString:
		return "String"
	case OperandNumber:
		return "Number"
	case OperandArray:
		return "Array"
	case OperandName:
		return "Name"
	case OperandRef:
		return "Ref"
	default:
		return "Unknown"
	}
}

// ContentOperand represents a single operand for a content stream operator.
// The Kind field determines which value field is valid.
type ContentOperand struct {
	// Kind identifies the type of this operand.
	Kind OperandKind
	// StrVal holds the value for String or Name operands.
	StrVal string
	// NumVal holds the value for Number operands.
	NumVal float64
	// ArrVal holds the elements for Array operands.
	ArrVal []ContentOperand
	// RefVal holds the indirect reference for Ref operands (format: "N G R").
	RefVal string
}

// String returns a string representation of the operand for debugging.
func (op ContentOperand) String() string {
	switch op.Kind {
	case OperandString:
		return fmt.Sprintf("(%s)", op.StrVal)
	case OperandName:
		return fmt.Sprintf("/%s", op.StrVal)
	case OperandNumber:
		if op.NumVal == float64(int64(op.NumVal)) {
			return fmt.Sprintf("%d", int64(op.NumVal))
		}
		return fmt.Sprintf("%g", op.NumVal)
	case OperandArray:
		var parts []string
		for _, elem := range op.ArrVal {
			parts = append(parts, elem.String())
		}
		return fmt.Sprintf("[%s]", strings.Join(parts, " "))
	case OperandRef:
		return op.RefVal
	default:
		return "?"
	}
}

// IsNumber returns true if the operand is a number with the specified value.
func (op ContentOperand) IsNumber(val float64) bool {
	return op.Kind == OperandNumber && op.NumVal == val
}

// IsString returns true if the operand is a string with the specified value.
func (op ContentOperand) IsString(val string) bool {
	return op.Kind == OperandString && op.StrVal == val
}

// ContentOperator represents a single PDF content stream operator and its operands.
type ContentOperator struct {
	// Op is the operator name (e.g., "Tj", "TJ", "BT", "ET").
	Op string
	// Operands holds the arguments preceding the operator, in order.
	Operands []ContentOperand
}

// String returns a string representation for debugging.
func (co ContentOperator) String() string {
	var parts []string
	for _, op := range co.Operands {
		parts = append(parts, op.String())
	}
	if len(parts) > 0 {
		return fmt.Sprintf("%s %s", strings.Join(parts, " "), co.Op)
	}
	return co.Op
}

// contentLexer provides lexical analysis for PDF content streams.
type contentLexer struct {
	data   []byte
	pos    int
	length int
}

// newContentLexer creates a new lexer for the given content stream data.
func newContentLexer(data []byte) *contentLexer {
	return &contentLexer{
		data:   data,
		pos:    0,
		length: len(data),
	}
}

// peek returns the current byte without advancing, or 0 if at end.
func (l *contentLexer) peek() byte {
	if l.pos >= l.length {
		return 0
	}
	return l.data[l.pos]
}

// advance returns the current byte and advances the position.
func (l *contentLexer) advance() byte {
	if l.pos >= l.length {
		return 0
	}
	ch := l.data[l.pos]
	l.pos++
	return ch
}

// skipWhitespace skips spaces, newlines, and comments.
func (l *contentLexer) skipWhitespace() {
	for l.pos < l.length {
		ch := l.peek()
		// PDF whitespace characters: space, tab, CR, LF, NUL, form feed
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' || ch == '\x00' || ch == '\f' {
			l.advance()
		} else if ch == '%' {
			// Skip comment: everything until newline (LF or CR)
			for l.pos < l.length && l.peek() != '\n' && l.peek() != '\r' {
				l.advance()
			}
		} else {
			break
		}
	}
}

// isOperatorChar returns true for characters that can form operator names.
// PDF operators are single letter commands or combinations like Tj, TJ, etc.
func isOperatorChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '*' || ch == '\'' || ch == '"'
}

// readLiteralString reads a PDF literal string in parentheses.
// Handles nested parentheses and escape sequences.
func (l *contentLexer) readLiteralString() (string, error) {
	if l.peek() != '(' {
		return "", fmt.Errorf("expected '(' at position %d", l.pos)
	}
	l.advance() // consume '('

	var result strings.Builder
	depth := 1

	for l.pos < l.length && depth > 0 {
		ch := l.advance()
		if ch == '\\' && l.pos < l.length {
			// Handle escape sequences
			next := l.peek()
			switch next {
			case 'n':
				result.WriteByte('\n')
				l.advance()
			case 'r':
				result.WriteByte('\r')
				l.advance()
			case 't':
				result.WriteByte('\t')
				l.advance()
			case 'b':
				result.WriteByte('\b')
				l.advance()
			case 'f':
				result.WriteByte('\f')
				l.advance()
			case '(', ')', '\\':
				result.WriteByte(next)
				l.advance()
			default:
				// Octal escape sequence (up to 3 digits)
				if next >= '0' && next <= '7' {
					octal := 0
					for i := 0; i < 3 && l.pos < l.length; i++ {
						c := l.peek()
						if c < '0' || c > '7' {
							break
						}
						octal = octal*8 + int(c-'0')
						l.advance()
					}
					result.WriteByte(byte(octal))
				} else {
					result.WriteByte(next)
					l.advance()
				}
			}
		} else if ch == '(' {
			depth++
			result.WriteByte(ch)
		} else if ch == ')' {
			depth--
			if depth > 0 {
				result.WriteByte(ch)
			}
		} else {
			result.WriteByte(ch)
		}
	}

	if depth != 0 {
		return result.String(), fmt.Errorf("unterminated literal string")
	}

	return result.String(), nil
}

// readHexString reads a PDF hex string in angle brackets.
func (l *contentLexer) readHexString() (string, error) {
	if l.peek() != '<' {
		return "", fmt.Errorf("expected '<' at position %d", l.pos)
	}
	l.advance() // consume '<'

	var hexChars strings.Builder
	for l.pos < l.length && l.peek() != '>' {
		ch := l.peek()
		if isHexDigit(ch) {
			hexChars.WriteByte(l.advance())
		} else if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance() // skip whitespace in hex strings
		} else {
			return "", fmt.Errorf("invalid character in hex string at position %d", l.pos)
		}
	}

	if l.pos >= l.length {
		return "", fmt.Errorf("unterminated hex string")
	}

	l.advance() // consume '>'

	hexStr := hexChars.String()
	if len(hexStr)%2 != 0 {
		hexStr += "0" // pad with zero if odd
	}

	// Decode hex string
	data, err := hexDecodeString(hexStr)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// isHexDigit returns true if ch is a hexadecimal digit.
func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'F') || (ch >= 'a' && ch <= 'f')
}

// hexDecodeString decodes a hex string to bytes.
func hexDecodeString(hexStr string) ([]byte, error) {
	result := make([]byte, len(hexStr)/2)
	for i := 0; i < len(hexStr); i += 2 {
		b, err := strconv.ParseUint(hexStr[i:i+2], 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex at position %d: %w", i, err)
		}
		result[i/2] = byte(b)
	}
	return result, nil
}

// readName reads a PDF name starting with '/'.
func (l *contentLexer) readName() (string, error) {
	if l.peek() != '/' {
		return "", fmt.Errorf("expected '/' at position %d", l.pos)
	}
	l.advance() // consume '/'

	var result strings.Builder
	for l.pos < l.length {
		ch := l.peek()
		// Names end at whitespace or delimiter characters
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' || ch == '\x00' || ch == '\f' {
			break
		}
		if ch == '(' || ch == ')' || ch == '<' || ch == '>' || ch == '[' || ch == ']' || ch == '{' || ch == '}' || ch == '/' || ch == '%' {
			break
		}
		if ch == '#' && l.pos+2 < l.length {
			// Hex escape in name: #XX
			hexCode := string(l.data[l.pos+1 : l.pos+3])
			if isHexDigit(l.data[l.pos+1]) && isHexDigit(l.data[l.pos+2]) {
				val, _ := strconv.ParseUint(hexCode, 16, 8)
				result.WriteByte(byte(val))
				l.pos += 3
				continue
			}
		}
		result.WriteByte(l.advance())
	}

	if result.Len() == 0 {
		return "", fmt.Errorf("empty name at position %d", l.pos)
	}

	return result.String(), nil
}

// readNumber reads an integer or real number.
func (l *contentLexer) readNumber() (float64, error) {
	start := l.pos
	// Optional sign
	if l.peek() == '-' || l.peek() == '+' {
		l.advance()
	}

	// Integer part
	hasDigits := false
	for l.pos < l.length && l.peek() >= '0' && l.peek() <= '9' {
		l.advance()
		hasDigits = true
	}

	// Decimal part
	if l.peek() == '.' {
		l.advance()
		for l.pos < l.length && l.peek() >= '0' && l.peek() <= '9' {
			l.advance()
			hasDigits = true
		}
	}

	if !hasDigits {
		return 0, fmt.Errorf("expected number at position %d", start)
	}

	numStr := string(l.data[start:l.pos])
	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number %q at position %d: %w", numStr, start, err)
	}

	return val, nil
}

// readArray reads a PDF array [...].
func (l *contentLexer) readArray() ([]ContentOperand, error) {
	if l.peek() != '[' {
		return nil, fmt.Errorf("expected '[' at position %d", l.pos)
	}
	l.advance() // consume '['

	var elements []ContentOperand
	for l.pos < l.length {
		l.skipWhitespace()
		if l.pos >= l.length || l.peek() == ']' {
			break
		}

		elem, err := l.readOperand()
		if err != nil {
			return nil, err
		}
		elements = append(elements, elem)
	}

	if l.pos >= l.length || l.peek() != ']' {
		return nil, fmt.Errorf("unterminated array")
	}
	l.advance() // consume ']'

	return elements, nil
}

// readOperand reads a single operand based on its prefix.
func (l *contentLexer) readOperand() (ContentOperand, error) {
	l.skipWhitespace()
	if l.pos >= l.length {
		return ContentOperand{}, fmt.Errorf("unexpected end of content stream")
	}

	ch := l.peek()

	switch ch {
	case '(':
		str, err := l.readLiteralString()
		if err != nil {
			return ContentOperand{}, err
		}
		return ContentOperand{Kind: OperandString, StrVal: str}, nil

	case '<':
		// Check for "<<" dictionary marker vs "<" hex string
		if l.pos+1 < l.length && l.data[l.pos+1] == '<' {
			return ContentOperand{}, fmt.Errorf("dictionaries not supported as operands")
		}
		str, err := l.readHexString()
		if err != nil {
			return ContentOperand{}, err
		}
		return ContentOperand{Kind: OperandString, StrVal: str}, nil

	case '/':
		name, err := l.readName()
		if err != nil {
			return ContentOperand{}, err
		}
		return ContentOperand{Kind: OperandName, StrVal: name}, nil

	case '[':
		arr, err := l.readArray()
		if err != nil {
			return ContentOperand{}, err
		}
		return ContentOperand{Kind: OperandArray, ArrVal: arr}, nil

	case '-', '+', '.', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		num, err := l.readNumber()
		if err != nil {
			return ContentOperand{}, err
		}
		return ContentOperand{Kind: OperandNumber, NumVal: num}, nil

	default:
		// Might be a reference "N G R" or unexpected token
		return ContentOperand{}, fmt.Errorf("unexpected character '%c' at position %d", ch, l.pos)
	}
}

// readOperator reads an operator name.
func (l *contentLexer) readOperator() (string, error) {
	l.skipWhitespace()
	if l.pos >= l.length {
		return "", fmt.Errorf("unexpected end of content stream")
	}

	start := l.pos
	// PDF operators are typically one or two letters, sometimes with special chars
	for l.pos < l.length && isOperatorChar(l.peek()) {
		l.advance()
	}

	if start == l.pos {
		return "", fmt.Errorf("expected operator at position %d", start)
	}

	return string(l.data[start:l.pos]), nil
}

// parseContentStream parses a PDF content stream into a sequence of operators.
// It handles strings (parentheses and hex), numbers, arrays, names, and operators.
func parseContentStream(data []byte) ([]ContentOperator, error) {
	lexer := newContentLexer(data)
	var operators []ContentOperator

	for {
		lexer.skipWhitespace()
		if lexer.pos >= lexer.length {
			break
		}

		ch := lexer.peek()

		// Check if this is the start of an operand or end of stream
		if ch == '(' || ch == '<' || ch == '/' || ch == '[' ||
			ch == '-' || ch == '+' || ch == '.' ||
			(ch >= '0' && ch <= '9') {
			// Read operand
			operand, err := lexer.readOperand()
			if err != nil {
				// If we can't parse as operand, try parsing as operator
				// Some operators like ' and " start with characters that look like numbers
				opStr, opErr := lexer.readOperator()
				if opErr != nil {
					// Try to recover by skipping one character
					lexer.advance()
					continue
				}
				// Check if this is a valid operator
				if isValidOperator(opStr) {
					operators = append(operators, ContentOperator{Op: opStr, Operands: nil})
				}
				continue
			}

			// Check if the next token is an operator or another operand
			// We need to peek ahead to determine this
			lexer.skipWhitespace()
			if lexer.pos >= lexer.length {
				// Trailing operand without operator - skip it
				continue
			}

			// Collect operands until we hit an operator
			operands := []ContentOperand{operand}

			for lexer.pos < lexer.length {
				lexer.skipWhitespace()
				if lexer.pos >= lexer.length {
					break
				}

				nextCh := lexer.peek()

				// Check if this looks like an operator (single letter or quoted)
				if isOperatorChar(nextCh) {
					// Try to read as operator
					pos := lexer.pos
					opStr, err := lexer.readOperator()
					if err == nil && isValidOperator(opStr) {
						// This is an operator, finalize the current operation
						operators = append(operators, ContentOperator{Op: opStr, Operands: operands})
						break
					}
					// Not a valid operator, reset and try as operand
					lexer.pos = pos
				}

				// Try to read as another operand
				nextOp, err := lexer.readOperand()
				if err != nil {
					// Can't parse as operand, skip this character and continue
					lexer.advance()
					continue
				}
				operands = append(operands, nextOp)
			}
		} else if isOperatorChar(ch) {
			// Read operator without operands
			opStr, err := lexer.readOperator()
			if err != nil {
				// Skip problematic character
				lexer.advance()
				continue
			}
			operators = append(operators, ContentOperator{Op: opStr, Operands: nil})
		} else {
			// Unknown character, skip it
			lexer.advance()
		}
	}

	return operators, nil
}

// isValidOperator returns true if s is a known PDF content stream operator.
// This list includes all text, graphics, and state operators.
func isValidOperator(s string) bool {
	// Common text operators
	textOps := map[string]bool{
		"BT": true, "ET": true, // Begin/End Text
		"Tj": true, "TJ": true, // Show text
		"'": true, "\"": true,  // Show text with line break
		"Tf": true,             // Set font
		"Tc": true,             // Set character spacing
		"Tw": true,             // Set word spacing
		"Tz": true,             // Set horizontal scaling
		"TL": true,             // Set leading
		"Ts": true,             // Set text rise
		"Tm": true,             // Set text matrix
		"T*": true,             // Move to next line
		"Td": true, "TD": true, // Move text position
		"Tr": true,             // Set rendering mode
	}

	// Graphics state operators
	stateOps := map[string]bool{
		"q": true, "Q": true,  // Save/Restore graphics state
		"cm": true,            // Concatenate matrix
		"w": true,             // Set line width
		"J": true,             // Set line cap
		"j": true,             // Set line join
		"M": true,             // Set miter limit
		"d": true,             // Set dash pattern
		"ri": true,            // Set rendering intent
		"i": true,             // Set flatness
		"gs": true,            // Set graphics state parameters
	}

	// Path construction operators
	pathOps := map[string]bool{
		"m": true, "l": true, "c": true, "v": true, "y": true, "h": true, // Move, Line, Curve, Close
		"re": true, // Rectangle
	}

	// Path painting operators
	paintOps := map[string]bool{
		"S": true, "s": true, "f": true, "F": true, "f*": true,
		"B": true, "B*": true, "b": true, "b*": true, "n": true,
	}

	// Color operators
	colorOps := map[string]bool{
		"CS": true, "cs": true, "SC": true, "SCN": true, "sc": true, "scn": true,
		"G": true, "g": true, "RG": true, "rg": true, "K": true, "k": true,
	}

	// Clipping operators
	clipOps := map[string]bool{
		"W": true, "W*": true,
	}

	// XObject and resource operators
	resourceOps := map[string]bool{
		"Do": true, // Draw XObject (image, form)
		"BI": true, "ID": true, "EI": true, // Inline image
	}

	// Marked content operators
	markOps := map[string]bool{
		"MP": true, "DP": true, "BMC": true, "BDC": true, "EMC": true,
	}

	// Compatibility operators
	compatOps := map[string]bool{
		"BX": true, "EX": true,
	}

	allOps := make(map[string]bool)
	for k, v := range textOps {
		allOps[k] = v
	}
	for k, v := range stateOps {
		allOps[k] = v
	}
	for k, v := range pathOps {
		allOps[k] = v
	}
	for k, v := range paintOps {
		allOps[k] = v
	}
	for k, v := range colorOps {
		allOps[k] = v
	}
	for k, v := range clipOps {
		allOps[k] = v
	}
	for k, v := range resourceOps {
		allOps[k] = v
	}
	for k, v := range markOps {
		allOps[k] = v
	}
	for k, v := range compatOps {
		allOps[k] = v
	}

	return allOps[s]
}

// extractTextFromOperators extracts text content from parsed content stream operators.
// It handles BT/ET text blocks, Tj/TJ text showing operators, and properly manages
// character spacing adjustments in TJ arrays.
func extractTextFromOperators(operators []ContentOperator) string {
	var result strings.Builder
	inTextBlock := false
	needsSpace := false

	for _, op := range operators {
		switch op.Op {
		case "BT":
			// Begin text block
			inTextBlock = true
			needsSpace = false

		case "ET":
			// End text block
			inTextBlock = false
			needsSpace = false

		case "Tj":
			// Show text: (string) Tj
			if !inTextBlock || len(op.Operands) == 0 {
				continue
			}
			if needsSpace && result.Len() > 0 {
				result.WriteByte(' ')
			}
			text := extractOperandString(op.Operands[0])
			result.WriteString(text)
			needsSpace = true

		case "TJ":
			// Show text array: [(string1) num1 (string2) ...] TJ
			if !inTextBlock || len(op.Operands) == 0 {
				continue
			}
			arr := op.Operands[0]
			if arr.Kind != OperandArray {
				continue
			}
			processTJArray(arr.ArrVal, &result, &needsSpace)

		case "'":
			// Move to next line and show text: (string) '
			if !inTextBlock || len(op.Operands) == 0 {
				continue
			}
			if result.Len() > 0 {
				result.WriteByte('\n')
			}
			text := extractOperandString(op.Operands[0])
			result.WriteString(text)
			needsSpace = true

		case "\"":
			// Set word/char spacing, move to next line, show text: wordSpace charSpace (string) "
			if !inTextBlock || len(op.Operands) < 3 {
				continue
			}
			if result.Len() > 0 {
				result.WriteByte('\n')
			}
			text := extractOperandString(op.Operands[2])
			result.WriteString(text)
			needsSpace = true

		case "T*":
			// Move to next line
			if inTextBlock && result.Len() > 0 {
				result.WriteByte('\n')
			}
			needsSpace = false

		case "Td", "TD":
			// Move text position
			if !inTextBlock || len(op.Operands) < 2 {
				continue
			}
			// Check if this is a vertical move indicating a new line
			tx := op.Operands[0].NumVal
			ty := op.Operands[1].NumVal
			// If y offset is significant, treat as new line
			if ty < 0 && result.Len() > 0 {
				result.WriteByte('\n')
			}
			// Small negative tx might indicate word spacing
			if tx < -50 && needsSpace && result.Len() > 0 {
				result.WriteByte(' ')
			}
			needsSpace = false
		}
	}

	return strings.TrimSpace(result.String())
}

// processTJArray processes a TJ array, handling character spacing adjustments.
// The array alternates between strings and numeric adjustments.
// A large negative adjustment (wordSpacingThreshold) indicates a word break.
const wordSpacingThreshold = -250.0 // thousandths of text space units

func processTJArray(elements []ContentOperand, result *strings.Builder, needsSpace *bool) {
	for i, elem := range elements {
		if elem.Kind == OperandString {
			// Check if previous element was a large negative spacing
			if *needsSpace && result.Len() > 0 {
				if i > 0 && elements[i-1].Kind == OperandNumber {
					adjustment := elements[i-1].NumVal
					// Large negative adjustment indicates word boundary
					if adjustment < wordSpacingThreshold {
						result.WriteByte(' ')
					}
				}
			}
			result.WriteString(elem.StrVal)
			*needsSpace = true
		}
		// Numeric elements are spacing adjustments, handled when we see the next string
	}
}

// extractOperandString extracts a string value from a ContentOperand.
// It handles UTF-16 encoding detection and decoding.
func extractOperandString(op ContentOperand) string {
	if op.Kind != OperandString {
		return ""
	}
	str := op.StrVal

	// Handle UTF-16 BOM detection
	if len(str) >= 2 {
		b := []byte(str)
		if b[0] == 0xFE && b[1] == 0xFF {
			return utf16BEToUTF8String(b[2:])
		}
		if b[0] == 0xFF && b[1] == 0xFE {
			return utf16LEToUTF8String(b[2:])
		}
		// Heuristic: no-BOM UTF-16BE detection
		if len(b) >= 4 && len(b)%2 == 0 {
			zeroCount := 0
			for i := 1; i < len(b); i += 2 {
				if b[i] == 0 {
					zeroCount++
				}
			}
			if zeroCount > len(b)/4 {
				return utf16BEToUTF8String(b)
			}
		}
	}

	return str
}

// utf16BEToUTF8String converts UTF-16BE bytes to a UTF-8 string.
func utf16BEToUTF8String(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data)-1; i += 2 {
		runes = append(runes, rune(data[i])<<8|rune(data[i+1]))
	}
	return string(runes)
}

// utf16LEToUTF8String converts UTF-16LE bytes to a UTF-8 string.
func utf16LEToUTF8String(data []byte) string {
	if len(data)%2 != 0 {
		data = data[:len(data)-1]
	}
	runes := make([]rune, 0, len(data)/2)
	for i := 0; i < len(data)-1; i += 2 {
		runes = append(runes, rune(data[i])|rune(data[i+1])<<8)
	}
	return string(runes)
}
