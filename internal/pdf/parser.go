package pdf

import (
	"fmt"
	"strconv"
	"unicode"
)

type PDFObject interface{}

type PDFName string

type PDFInteger int64

type PDFReal float64

type PDFRef struct {
	ObjNum int64
	GenNum int64
}

type PDFArray []PDFObject

type PDFHexString string

type PDFLiteralString string

type PDFBool bool

type PDFDict map[PDFName]PDFObject

func parsePDFName(input string) (PDFName, string, error) {
	if len(input) < 2 || input[0] != '/' {
		return "", "", fmt.Errorf("parsePDFName: input does not start with /")
	}
	i := 1
	for i < len(input) {
		ch := rune(input[i])
		if unicode.IsSpace(ch) || ch == '/' || ch == '<' || ch == '>' || ch == '[' || ch == ']' {
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

func DictGet(d PDFDict, key string) PDFObject {
	return d[PDFName(key)]
}

func DictGetName(d PDFDict, key string) (PDFName, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return "", false
	}
	name, ok := v.(PDFName)
	return name, ok
}

func DictGetRef(d PDFDict, key string) (PDFRef, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return PDFRef{}, false
	}
	ref, ok := v.(PDFRef)
	return ref, ok
}

func DictGetInt(d PDFDict, key string) (int64, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return 0, false
	}
	i, ok := v.(PDFInteger)
	return int64(i), ok
}

func DictGetArray(d PDFDict, key string) (PDFArray, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return nil, false
	}
	arr, ok := v.(PDFArray)
	return arr, ok
}

func DictGetDict(d PDFDict, key string) (PDFDict, bool) {
	v, ok := d[PDFName(key)]
	if !ok {
		return nil, false
	}
	dict, ok := v.(PDFDict)
	return dict, ok
}

func ArrayToRefs(arr PDFArray) []PDFRef {
	var refs []PDFRef
	for _, obj := range arr {
		if ref, ok := obj.(PDFRef); ok {
			refs = append(refs, ref)
		}
	}
	return refs
}

func RefToString(ref PDFRef) string {
	return fmt.Sprintf("%d %d R", ref.ObjNum, ref.GenNum)
}
