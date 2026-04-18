package pdf

import (
	"testing"
)

func TestParsePDFName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		rest     string
	}{
		{"/Type", "Type", ""},
		{"/Pages", "Pages", ""},
		{"/Kids[", "Kids", "["},
		{"/Count ", "Count", " "},
		{"/Type /Catalog", "Type", " /Catalog"},
	}

	for _, tt := range tests {
		name, rest, err := parsePDFName(tt.input)
		if err != nil {
			t.Fatalf("parsePDFName(%q): error = %v", tt.input, err)
		}
		if name != PDFName(tt.expected) {
			t.Fatalf("parsePDFName(%q): name = %q, want %q", tt.input, name, tt.expected)
		}
		if rest != tt.rest {
			t.Fatalf("parsePDFName(%q): rest = %q, want %q", tt.input, rest, tt.rest)
		}
	}
}

func TestParsePDFInteger(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		rest     string
	}{
		{"123", 123, ""},
		{"0", 0, ""},
		{"65535", 65535, ""},
		{"123 ", 123, " "},
		{"123\n", 123, "\n"},
		{"-10", -10, ""},
	}

	for _, tt := range tests {
		num, rest, err := parsePDFInteger(tt.input)
		if err != nil {
			t.Fatalf("parsePDFInteger(%q): error = %v", tt.input, err)
		}
		if num != PDFInteger(tt.expected) {
			t.Fatalf("parsePDFInteger(%q): num = %d, want %d", tt.input, num, tt.expected)
		}
		if rest != tt.rest {
			t.Fatalf("parsePDFInteger(%q): rest = %q, want %q", tt.input, rest, tt.rest)
		}
	}
}

func TestParsePDFRef(t *testing.T) {
	tests := []struct {
		input     string
		expected  PDFRef
		rest      string
		wantError bool
	}{
		{"3 0 R", PDFRef{3, 0}, "", false},
		{"1 0 R", PDFRef{1, 0}, "", false},
		{"100 50 R", PDFRef{100, 50}, "", false},
		{"3 0 R ", PDFRef{3, 0}, " ", false},
		{"3 0", PDFRef{}, "", true},
		{"3 R", PDFRef{}, "", true},
		{"R", PDFRef{}, "", true},
	}

	for _, tt := range tests {
		ref, rest, err := parsePDFRef(tt.input)
		if tt.wantError {
			if err == nil {
				t.Fatalf("parsePDFRef(%q): expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parsePDFRef(%q): error = %v", tt.input, err)
		}
		if ref != tt.expected {
			t.Fatalf("parsePDFRef(%q): ref = %v, want %v", tt.input, ref, tt.expected)
		}
		if rest != tt.rest {
			t.Fatalf("parsePDFRef(%q): rest = %q, want %q", tt.input, rest, tt.rest)
		}
	}
}

func TestParsePDFArray(t *testing.T) {
	tests := []struct {
		input    string
		expected []PDFObject
		rest     string
		wantErr  bool
	}{
		{
			input:    "[]",
			expected: []PDFObject{},
			rest:     "",
			wantErr:  false,
		},
		{
			input:    "[3 0 R]",
			expected: []PDFObject{PDFRef{3, 0}},
			rest:     "",
			wantErr:  false,
		},
		{
			input:    "[3 0 R 4 0 R]",
			expected: []PDFObject{PDFRef{3, 0}, PDFRef{4, 0}},
			rest:     "",
			wantErr:  false,
		},
		{
			input:    "[3 0 R] ",
			expected: []PDFObject{PDFRef{3, 0}},
			rest:     " ",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		arr, rest, err := parsePDFArray(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("parsePDFArray(%q): expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parsePDFArray(%q): error = %v", tt.input, err)
		}
		if len(arr) != len(tt.expected) {
			t.Fatalf("parsePDFArray(%q): len = %d, want %d", tt.input, len(arr), len(tt.expected))
		}
		for i, obj := range arr {
			if !objectsEqual(obj, tt.expected[i]) {
				t.Fatalf("parsePDFArray(%q)[%d] = %v, want %v", tt.input, i, obj, tt.expected[i])
			}
		}
		if rest != tt.rest {
			t.Fatalf("parsePDFArray(%q): rest = %q, want %q", tt.input, rest, tt.rest)
		}
	}
}

func TestParsePDFDict(t *testing.T) {
	tests := []struct {
		input    string
		expected PDFDict
		rest     string
		wantErr  bool
	}{
		{
			input:    "<< /Type /Catalog >>",
			expected: PDFDict{PDFName("Type"): PDFName("Catalog")},
			rest:     "",
			wantErr:  false,
		},
		{
			input: "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
			expected: PDFDict{
				PDFName("Type"):  PDFName("Pages"),
				PDFName("Kids"):  PDFArray{PDFRef{3, 0}},
				PDFName("Count"): PDFInteger(1),
			},
			rest:    "",
			wantErr: false,
		},
		{
			input:    "<< /Pages 2 0 R >>",
			expected: PDFDict{PDFName("Pages"): PDFRef{2, 0}},
			rest:     "",
			wantErr:  false,
		},
		{
			input:    "<< >>",
			expected: PDFDict{},
			rest:     "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		d, rest, err := parsePDFDict(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("parsePDFDict(%q): expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parsePDFDict(%q): error = %v", tt.input, err)
		}
		if len(d) != len(tt.expected) {
			t.Fatalf("parsePDFDict(%q): len = %d, want %d", tt.input, len(d), len(tt.expected))
		}
		for k, v := range tt.expected {
			if !objectsEqual(d[k], v) {
				t.Fatalf("parsePDFDict(%q)[%q] = %v, want %v", tt.input, k, d[k], v)
			}
		}
		if rest != tt.rest {
			t.Fatalf("parsePDFDict(%q): rest = %q, want %q", tt.input, rest, tt.rest)
		}
	}
}

func TestParsePDFObject(t *testing.T) {
	tests := []struct {
		input    string
		expected PDFObject
		wantErr  bool
	}{
		{"/Type", PDFName("Type"), false},
		{"123", PDFInteger(123), false},
		{"-10", PDFInteger(-10), false},
		{"3 0 R", PDFRef{3, 0}, false},
		{"[3 0 R]", PDFArray{PDFRef{3, 0}}, false},
		{"<< /Type /Page >>", PDFDict{PDFName("Type"): PDFName("Page")}, false},
		{"[<< /Type /Catalog >> 2 0 R]", PDFArray{PDFDict{PDFName("Type"): PDFName("Catalog")}, PDFRef{2, 0}}, false},
	}

	for _, tt := range tests {
		obj, _, err := parsePDFObject(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("parsePDFObject(%q): expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parsePDFObject(%q): error = %v", tt.input, err)
		}
		if !objectsEqual(obj, tt.expected) {
			t.Fatalf("parsePDFObject(%q) = %v, want %v", tt.input, obj, tt.expected)
		}
	}
}

func objectsEqual(a, b PDFObject) bool {
	switch av := a.(type) {
	case PDFName:
		bv, ok := b.(PDFName)
		return ok && av == bv
	case PDFInteger:
		bv, ok := b.(PDFInteger)
		return ok && av == bv
	case PDFRef:
		bv, ok := b.(PDFRef)
		return ok && av.ObjNum == bv.ObjNum && av.GenNum == bv.GenNum
	case PDFArray:
		bv, ok := b.(PDFArray)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !objectsEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case PDFDict:
		bv, ok := b.(PDFDict)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			if !objectsEqual(v, bv[k]) {
				return false
			}
		}
		return true
	}
	return false
}

func TestParseDictAndExtractRootRef(t *testing.T) {
	dictStr := "<< /Root 1 0 R /Size 4 >>"
	d, err := ParseDictContent(dictStr)
	if err != nil {
		t.Fatalf("ParseDictContent: %v", err)
	}
	ref, ok := DictGetRef(d, "Root")
	if !ok {
		t.Fatalf("DictGetRef(Root): not found")
	}
	if ref.ObjNum != 1 || ref.GenNum != 0 {
		t.Fatalf("Root ref = (%d, %d), want (1, 0)", ref.ObjNum, ref.GenNum)
	}
}

func TestParseDictAndExtractPagesRef(t *testing.T) {
	dictStr := "<< /Type /Catalog /Pages 2 0 R >>"
	d, err := ParseDictContent(dictStr)
	if err != nil {
		t.Fatalf("ParseDictContent: %v", err)
	}
	ref, ok := DictGetRef(d, "Pages")
	if !ok {
		t.Fatalf("DictGetRef(Pages): not found")
	}
	if ref.ObjNum != 2 || ref.GenNum != 0 {
		t.Fatalf("Pages ref = (%d, %d), want (2, 0)", ref.ObjNum, ref.GenNum)
	}
}

func TestParseDictAndExtractPagesRefFromArray(t *testing.T) {
	dictStr := "<< /Type /Catalog /Pages [2 0 R] >>"
	d, err := ParseDictContent(dictStr)
	if err != nil {
		t.Fatalf("ParseDictContent: %v", err)
	}
	arr, ok := DictGetArray(d, "Pages")
	if !ok {
		t.Fatalf("DictGetArray(Pages): not found")
	}
	if len(arr) != 1 {
		t.Fatalf("Pages array len = %d, want 1", len(arr))
	}
	ref, ok := arr[0].(PDFRef)
	if !ok {
		t.Fatalf("Pages[0] is not PDFRef")
	}
	if ref.ObjNum != 2 || ref.GenNum != 0 {
		t.Fatalf("Pages ref = (%d, %d), want (2, 0)", ref.ObjNum, ref.GenNum)
	}
}

func TestParseDictAndExtractKidsAndCount(t *testing.T) {
	dictStr := "<< /Type /Pages /Kids [3 0 R 4 0 R] /Count 2 >>"
	d, err := ParseDictContent(dictStr)
	if err != nil {
		t.Fatalf("ParseDictContent: %v", err)
	}

	arr, ok := DictGetArray(d, "Kids")
	if !ok {
		t.Fatalf("DictGetArray(Kids): not found")
	}
	refs := ArrayToRefs(arr)
	if len(refs) != 2 {
		t.Fatalf("Kids refs len = %d, want 2", len(refs))
	}
	if refs[0].ObjNum != 3 || refs[1].ObjNum != 4 {
		t.Fatalf("Kids refs = %v, want [3 0 R, 4 0 R]", refs)
	}

	cnt, ok := DictGetInt(d, "Count")
	if !ok {
		t.Fatalf("DictGetInt(Count): not found")
	}
	if cnt != 2 {
		t.Fatalf("Count = %d, want 2", cnt)
	}
}

func TestParseTrailerDictContent(t *testing.T) {
	dictStr := "<< /Root 1 0 R /Size 5 >>"
	d, err := ParseDictContent(dictStr)
	if err != nil {
		t.Fatalf("ParseDictContent: %v", err)
	}

	ref, ok := DictGetRef(d, "Root")
	if !ok {
		t.Fatalf("DictGetRef(Root): not found")
	}
	if RefToString(ref) != "1 0 R" {
		t.Fatalf("RefToString = %q, want %q", RefToString(ref), "1 0 R")
	}

	size, ok := DictGetInt(d, "Size")
	if !ok {
		t.Fatalf("DictGetInt(Size): not found")
	}
	if size != 5 {
		t.Fatalf("Size = %d, want 5", size)
	}
}

func TestParsePageDictAndExtractTypePage(t *testing.T) {
	dictStr := "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] >>"
	d, err := ParseDictContent(dictStr)
	if err != nil {
		t.Fatalf("ParseDictContent: %v", err)
	}

	typ, ok := DictGetName(d, "Type")
	if !ok {
		t.Fatalf("DictGetName(Type): not found")
	}
	if typ != "Page" {
		t.Fatalf("Type = %q, want %q", typ, "Page")
	}

	parent, ok := DictGetRef(d, "Parent")
	if !ok {
		t.Fatalf("DictGetRef(Parent): not found")
	}
	if parent.ObjNum != 2 || parent.GenNum != 0 {
		t.Fatalf("Parent ref = (%d, %d), want (2, 0)", parent.ObjNum, parent.GenNum)
	}

	mediaBox, ok := DictGetArray(d, "MediaBox")
	if !ok {
		t.Fatalf("DictGetArray(MediaBox): not found")
	}
	if len(mediaBox) != 4 {
		t.Fatalf("MediaBox len = %d, want 4", len(mediaBox))
	}
	for i, v := range mediaBox {
		iv, ok := v.(PDFInteger)
		if !ok {
			t.Fatalf("MediaBox[%d] is not PDFInteger", i)
		}
		_ = int64(iv)
	}
}

func TestParsePageDictAndValidateTypeNotPage(t *testing.T) {
	dictStr := "<< /Type /Action /S /JavaScript >>"
	d, err := ParseDictContent(dictStr)
	if err != nil {
		t.Fatalf("ParseDictContent: %v", err)
	}

	typ, ok := DictGetName(d, "Type")
	if !ok {
		t.Fatalf("DictGetName(Type): not found")
	}
	if typ == "Page" {
		t.Fatalf("Type = %q, should not be Page", typ)
	}
}

func TestParsePageDictMissingType(t *testing.T) {
	dictStr := "<< /Parent 2 0 R /MediaBox [0 0 612 792] >>"
	d, err := ParseDictContent(dictStr)
	if err != nil {
		t.Fatalf("ParseDictContent: %v", err)
	}

	typ, ok := DictGetName(d, "Type")
	if ok {
		t.Fatalf("DictGetName(Type): should not be present, got %q", typ)
	}
}
