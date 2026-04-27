package pdf

import (
	"fmt"
	"strconv"
	"unicode"
)

// PDFObject is the common interface for all PDF data types.
// It is used as a sum type for values parsed from PDF files.
type PDFObject interface{}

// PDFName represents a PDF name object (e.g., /Type, /Root).
// Names are atomic identifiers used as dictionary keys and enumeration values.
type PDFName string

// PDFInteger represents a PDF integer numeric object.
type PDFInteger int64

// PDFReal represents a PDF real (floating-point) numeric object.
type PDFReal float64

// PDFRef represents a PDF indirect reference (e.g., "12 0 R").
// It identifies an object by its object number and generation number.
type PDFRef struct {
	// ObjNum is the object number (positive integer) that uniquely identifies
	// the object within the PDF file.
	ObjNum int64
	// GenNum is the generation number, typically 0 for new objects.
	// Non-zero values indicate objects that have been replaced in incremental updates.
	GenNum int64
}

// PDFArray represents a PDF array object (ordered collection of PDF objects).
type PDFArray []PDFObject

// PDFHexString represents a PDF hex string literal (e.g., <4A6F6E>).
// The content is stored as raw characters, not decoded bytes.
type PDFHexString string

// PDFLiteralString represents a PDF literal string enclosed in parentheses.
// The content is stored without the outer parentheses but retains escape sequences.
type PDFLiteralString string

// PDFBool represents a PDF boolean value (true or false).
type PDFBool bool

// PDFNull represents the PDF null object (ISO 32000-2 §7.3.9).
// It is used for null array/dictionary values and absent optional entries.
type PDFNull struct{}

// PDFDict represents a PDF dictionary object (key-value map).
// Keys are PDFName and values are any PDFObject.
type PDFDict map[PDFName]PDFObject

func parsePDFName(input string) (PDFName, string, error) {
	if len(input) < 2 || input[0] != '/' {
		return "", "", fmt.Errorf("parsePDFName: input does not start with /")
	}
	i := 1
	for i < len(input) {
		ch := rune(input[i])
		if unicode.IsSpace(ch) || ch == '/' || ch == '<' || ch == '>' || ch == '[' || ch == ']' || ch == '(' || ch == ')' || ch == '{' || ch == '}' || ch == '%' {
			break
		}
		i++
	}
	return PDFName(input[1:i]), input[i:], nil
}

func parsePDFInteger(input string) (PDFInteger, string, error) {
	i := 0
	if len(input) > 0 && input[0] == '-' {
		i = 1
	}
	if i >= len(input) {
		return 0, "", fmt.Errorf("parsePDFInteger: no digits")
	}
	start := i
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		i++
	}
	if i == start {
		return 0, "", fmt.Errorf("parsePDFInteger: no digits found")
	}
	val, err := strconv.ParseInt(input[:i], 10, 64)
	if err != nil {
		return 0, "", err
	}
	return PDFInteger(val), input[i:], nil
}

func parsePDFReal(input string) (PDFReal, string, error) {
	i := 0
	if len(input) > 0 && input[0] == '-' {
		i = 1
	}
	if i >= len(input) {
		return 0, "", fmt.Errorf("parsePDFReal: no digits")
	}
	hasDigit := false
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		hasDigit = true
		i++
	}
	if i < len(input) && input[i] == '.' {
		i++
		for i < len(input) && input[i] >= '0' && input[i] <= '9' {
			hasDigit = true
			i++
		}
	}
	if !hasDigit {
		return 0, "", fmt.Errorf("parsePDFReal: no digits found")
	}
	val, err := strconv.ParseFloat(input[:i], 64)
	if err != nil {
		return 0, "", err
	}
	return PDFReal(val), input[i:], nil
}

func parsePDFHexString(input string) (PDFHexString, string, error) {
	if len(input) < 2 || input[0] != '<' {
		return "", "", fmt.Errorf("parsePDFHexString: does not start with <")
	}
	i := 1
	for i < len(input) && input[i] != '>' {
		i++
	}
	if i >= len(input) {
		return "", "", fmt.Errorf("parsePDFHexString: unclosed hex string")
	}
	hexContent := input[1:i]
	if len(hexContent)%2 != 0 {
		return "", "", fmt.Errorf("parsePDFHexString: odd number of hex digits")
	}
	for _, c := range hexContent {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') || (c >= 'a' && c <= 'f')) {
			return "", "", fmt.Errorf("parsePDFHexString: invalid hex digit")
		}
	}
	return PDFHexString(hexContent), input[i+1:], nil
}

func parsePDFLiteralString(input string) (PDFLiteralString, string, error) {
	if len(input) < 2 || input[0] != '(' {
		return "", "", fmt.Errorf("parsePDFLiteralString: does not start with (")
	}
	i := 1
	depth := 1
	for i < len(input) && depth > 0 {
		if input[i] == '\\' && i+1 < len(input) {
			i += 2
			continue
		}
		if input[i] == '(' {
			depth++
		} else if input[i] == ')' {
			depth--
		}
		i++
	}
	if depth > 0 {
		return "", "", fmt.Errorf("parsePDFLiteralString: unclosed string")
	}
	return PDFLiteralString(input[1 : i-1]), input[i:], nil
}

func parsePDFRef(input string) (PDFRef, string, error) {
	var ref PDFRef
	var err error

	ref.ObjNum, input, err = parseInt(input)
	if err != nil {
		return PDFRef{}, "", fmt.Errorf("parsePDFRef: objNum: %w", err)
	}

	input = skipSpace(input)
	ref.GenNum, input, err = parseInt(input)
	if err != nil {
		return PDFRef{}, "", fmt.Errorf("parsePDFRef: genNum: %w", err)
	}

	input = skipSpace(input)
	if len(input) < 1 || input[0] != 'R' {
		return PDFRef{}, "", fmt.Errorf("parsePDFRef: missing R")
	}

	return ref, input[1:], nil
}

func parseInt(input string) (int64, string, error) {
	input = skipSpace(input)
	i := 0
	if len(input) > 0 && input[0] == '-' {
		i = 1
	}
	if i >= len(input) {
		return 0, "", fmt.Errorf("no digits")
	}
	start := i
	for i < len(input) && input[i] >= '0' && input[i] <= '9' {
		i++
	}
	if i == start {
		return 0, "", fmt.Errorf("no digits")
	}
	val, err := strconv.ParseInt(input[:i], 10, 64)
	if err != nil {
		return 0, "", err
	}
	return val, input[i:], nil
}

func skipSpace(input string) string {
	i := 0
	for i < len(input) {
		if input[i] == '%' {
			for i < len(input) && input[i] != '\n' && input[i] != '\r' {
				i++
			}
			continue
		}
		if !unicode.IsSpace(rune(input[i])) {
			break
		}
		i++
	}
	return input[i:]
}

func parsePDFArray(input string) (PDFArray, string, error) {
	input = skipSpace(input)
	if len(input) < 2 || input[0] != '[' {
		return nil, "", fmt.Errorf("parsePDFArray: does not start with [")
	}
	input = input[1:]

	var arr PDFArray
	for {
		input = skipSpace(input)
		if len(input) == 0 {
			return nil, "", fmt.Errorf("parsePDFArray: unclosed array")
		}
		if input[0] == ']' {
			return arr, input[1:], nil
		}

		obj, rest, err := parsePDFObject(input)
		if err != nil {
			return nil, "", fmt.Errorf("parsePDFArray: element: %w", err)
		}
		arr = append(arr, obj)
		input = skipSpace(rest)
	}
}

func parsePDFDict(input string) (PDFDict, string, error) {
	input = skipSpace(input)
	if len(input) < 2 || input[0] != '<' || input[1] != '<' {
		return nil, "", fmt.Errorf("parsePDFDict: does not start with <<")
	}
	input = input[2:]

	dict := make(PDFDict)
	for {
		input = skipSpace(input)
		if len(input) >= 2 && input[0] == '>' && input[1] == '>' {
			return dict, input[2:], nil
		}
		if len(input) == 0 {
			return nil, "", fmt.Errorf("parsePDFDict: unclosed dict")
		}

		name, rest, err := parsePDFName(input)
		if err != nil {
			return nil, "", fmt.Errorf("parsePDFDict: name: %w", err)
		}
		input = skipSpace(rest)

		value, rest, err := parsePDFObject(input)
		if err != nil {
			return nil, "", fmt.Errorf("parsePDFDict: value for %s: %w", name, err)
		}
		dict[name] = value
		input = skipSpace(rest)
	}
}

func parsePDFObject(input string) (PDFObject, string, error) {
	input = skipSpace(input)
	if len(input) == 0 {
		return nil, "", fmt.Errorf("parsePDFObject: empty input")
	}

	switch input[0] {
	case '/':
		return parsePDFName(input)
	case '<':
		if len(input) > 1 && input[1] == '<' {
			return parsePDFDict(input)
		}
		return parsePDFHexString(input)
	case '[':
		return parsePDFArray(input)
	case '(':
		return parsePDFLiteralString(input)
	case 't':
		if len(input) >= 4 && input[:4] == "true" {
			return PDFBool(true), input[4:], nil
		}
		return nil, "", fmt.Errorf("parsePDFObject: unexpected token %q", string(input[0]))
	case 'f':
		if len(input) >= 5 && input[:5] == "false" {
			return PDFBool(false), input[5:], nil
		}
		return nil, "", fmt.Errorf("parsePDFObject: unexpected token %q", string(input[0]))
	case 'n':
		if len(input) >= 4 && input[:4] == "null" {
			return PDFNull{}, input[4:], nil
		}
		return nil, "", fmt.Errorf("parsePDFObject: unexpected token %q", string(input[0]))
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
		if input[0] == '-' {
			if len(input) > 1 && input[1] == '.' {
				return parsePDFReal(input)
			}
		}
		if input[0] != '-' {
			for i := 0; i < len(input) && input[i] >= '0' && input[i] <= '9'; i++ {
				if i+1 < len(input) && input[i+1] == '.' {
					return parsePDFReal(input)
				}
			}
		}
		objNum, rest, err := parseInt(input)
		if err != nil {
			return nil, "", err
		}
		rest = skipSpace(rest)
		if len(rest) > 0 && (rest[0] >= '0' && rest[0] <= '9' || rest[0] == '-') {
			genNum, rest2, err := parseInt(rest)
			if err != nil {
				return PDFInteger(objNum), rest, nil
			}
			rest2 = skipSpace(rest2)
			if len(rest2) > 0 && rest2[0] == 'R' {
				return PDFRef{objNum, genNum}, rest2[1:], nil
			}
			return PDFInteger(objNum), rest, nil
		}
		return PDFInteger(objNum), rest, nil
	case '.':
		return parsePDFReal(input)
	default:
		return nil, "", fmt.Errorf("parsePDFObject: unexpected token %q", string(input[0]))
	}
}

// ParseDictContent parses a PDF dictionary from its string representation.
// It expects the input to start with "<<" and returns the parsed PDFDict.
// Returns an error if the input is not a valid PDF dictionary.
func ParseDictContent(content string) (PDFDict, error) {
	obj, _, err := parsePDFObject(content)
	if err != nil {
		return nil, err
	}
	d, ok := obj.(PDFDict)
	if !ok {
		return nil, fmt.Errorf("ParseDictContent: not a dict")
	}
	return d, nil
}

// ParseArrayContent parses a PDF array from its string representation.
// It expects the input to start with "[" and returns the parsed PDFArray.
// Returns an error if the input is not a valid PDF array.
func ParseArrayContent(content string) (PDFArray, error) {
	obj, _, err := parsePDFObject(content)
	if err != nil {
		return nil, err
	}
	arr, ok := obj.(PDFArray)
	if !ok {
		return nil, fmt.Errorf("ParseArrayContent: not an array")
	}
	return arr, nil
}

// DictGet retrieves a value from the PDF dictionary by key.
// Returns nil if the key does not exist.
func DictGet(d PDFDict, key string) PDFObject {
	return d[PDFName(key)]
}

// DictGetName retrieves a PDF name value from the dictionary by key.
// The second return value reports whether the key exists and is of type PDFName.
func DictGetName(d PDFDict, key string) (PDFName, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return "", false
	}
	name, ok := v.(PDFName)
	return name, ok
}

// DictGetRef retrieves an indirect reference (PDFRef) from the dictionary by key.
// The second return value reports whether the key exists and is of type PDFRef.
func DictGetRef(d PDFDict, key string) (PDFRef, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return PDFRef{}, false
	}
	ref, ok := v.(PDFRef)
	return ref, ok
}

// DictGetInt retrieves an integer value from the dictionary by key.
// The second return value reports whether the key exists and is of type PDFInteger.
func DictGetInt(d PDFDict, key string) (int64, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return 0, false
	}
	i, ok := v.(PDFInteger)
	return int64(i), ok
}

// DictGetArray retrieves an array value from the dictionary by key.
// The second return value reports whether the key exists and is of type PDFArray.
func DictGetArray(d PDFDict, key string) (PDFArray, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return nil, false
	}
	arr, ok := v.(PDFArray)
	return arr, ok
}

// DictGetString retrieves a string value (PDFLiteralString or PDFHexString) from the dictionary.
// The second return value reports whether the key exists and is of a string type.
func DictGetString(d PDFDict, key string) (string, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return "", false
	}
	switch s := v.(type) {
	case PDFLiteralString:
		return string(s), true
	case PDFHexString:
		return string(s), true
	}
	return "", false
}

// DictGetDict retrieves a nested dictionary value from the dictionary by key.
// The second return value reports whether the key exists and is of type PDFDict.
func DictGetDict(d PDFDict, key string) (PDFDict, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return nil, false
	}
	dict, ok := v.(PDFDict)
	return dict, ok
}

// ArrayToRefs extracts all indirect references (PDFRef) from a PDF array.
// Non-reference elements are silently skipped.
func ArrayToRefs(arr PDFArray) []PDFRef {
	var refs []PDFRef
	for _, obj := range arr {
		if ref, ok := obj.(PDFRef); ok {
			refs = append(refs, ref)
		}
	}
	return refs
}

// RefToString formats a PDF indirect reference as its canonical string representation
// (e.g., "12 0 R" for object 12, generation 0).
func RefToString(ref PDFRef) string {
	return fmt.Sprintf("%d %d R", ref.ObjNum, ref.GenNum)
}
