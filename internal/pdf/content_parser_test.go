package pdf

import (
	"context"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
	testfixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

// TestParseContentStream tests the content stream parser with various PDF content stream constructs.
func TestParseContentStream(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantOps  int    // expected number of operators
		wantErr  bool   // whether we expect an error
		checkOp  string // specific operator to check for
	}{
		{
			name:    "simple Tj operator",
			input:   "(Hello World) Tj",
			wantOps: 1,
			checkOp: "Tj",
		},
		{
			name:    "TJ operator with spacing",
			input:   "[(H) -100 (ello) 200 (World)] TJ",
			wantOps: 1,
			checkOp: "TJ",
		},
		{
			name:    "BT/ET text block",
			input:   "BT /F1 12 Tf (text) Tj ET",
			wantOps: 4, // BT, Tf, Tj, ET
			checkOp: "BT",
		},
		{
			name:    "mixed graphics and text",
			input:   "10 20 30 40 re BT (text) Tj ET",
			wantOps: 4, // re, BT, Tj, ET
			checkOp: "re",
		},
		{
			name:    "text with line breaks",
			input:   "BT (Line 1) Tj T* (Line 2) Tj ET",
			wantOps: 5, // BT, Tj, T*, Tj, ET
			checkOp: "T*",
		},
		{
			name:    "quoted string operators",
			input:   "BT (text)' (more text)\" ET",
			wantOps: 4, // BT, ', ", ET
			checkOp: "'",
		},
		{
			name:    "comment handling",
			input:   "BT % this is a comment\n(text) Tj ET",
			wantOps: 3,
			checkOp: "Tj",
		},
		{
			name:    "hex string",
			input:   "<48656C6C6F> Tj",
			wantOps: 1,
			checkOp: "Tj",
		},
		{
			name:    "font setting",
			input:   "/F1 12 Tf",
			wantOps: 1,
			checkOp: "Tf",
		},
		{
			name:    "text positioning",
			input:   "100 200 Td",
			wantOps: 1,
			checkOp: "Td",
		},
		{
			name:    "text matrix",
			input:   "1 0 0 1 100 200 Tm",
			wantOps: 1,
			checkOp: "Tm",
		},
		{
			name:    "empty content stream",
			input:   "",
			wantOps: 0,
		},
		{
			name:    "whitespace only",
			input:   "   \n\t\r   ",
			wantOps: 0,
		},
		{
			name:    "array with hex string",
			input:   "[<48656C6C6F> 100] TJ",
			wantOps: 1,
			checkOp: "TJ",
		},
		{
			name:    "array with literal string",
			input:   "[(Hello) 100 (World)] TJ",
			wantOps: 1,
			checkOp: "TJ",
		},
		{
			name:    "array with mixed strings and numbers",
			input:   "[(H) -100 (ello) 200 (World) 50 <416C6C>] TJ",
			wantOps: 1,
			checkOp: "TJ",
		},
		{
			name:    "nested array with strings",
			input:   "[[(Inner) 50] 100 (Outer)] TJ",
			wantOps: 1,
			checkOp: "TJ",
		},
		{
			name:    "hex string as direct Tj operand",
			input:   "<416C6C6F> Tj",
			wantOps: 1,
			checkOp: "Tj",
		},
		{
			name:    "array with hex string spacing",
			input:   "[<48656C6C6F> 50 <576F726C64>] TJ",
			wantOps: 1,
			checkOp: "TJ",
		},
		{
			name:    "empty array",
			input:   "[] TJ",
			wantOps: 1,
			checkOp: "TJ",
		},
		{
			name:    "array with name and string",
			input:   "[/F1 (Text) 100] TJ",
			wantOps: 1,
			checkOp: "TJ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops, err := parseContentStream([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseContentStream() expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("parseContentStream() unexpected error: %v", err)
				return
			}

			if len(ops) != tt.wantOps {
				t.Errorf("parseContentStream() got %d operators, want %d", len(ops), tt.wantOps)
			}

			if tt.checkOp != "" {
				found := false
				for _, op := range ops {
					if op.Op == tt.checkOp {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("parseContentStream() did not find expected operator %q", tt.checkOp)
				}
			}
		})
	}
}

// TestExtractTextFromOperators tests the text extraction from parsed operators.
func TestExtractTextFromOperators(t *testing.T) {
	tests := []struct {
		name    string
		ops     []ContentOperator
		want    string
	}{
		{
			name: "single Tj string",
			ops: []ContentOperator{
				{Op: "BT", Operands: nil},
				{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "Hello"}}},
				{Op: "ET", Operands: nil},
			},
			want: "Hello",
		},
		{
			name: "TJ array no spacing",
			ops: []ContentOperator{
				{Op: "BT", Operands: nil},
				{Op: "TJ", Operands: []ContentOperand{{
					Kind: OperandArray,
					ArrVal: []ContentOperand{
						{Kind: OperandString, StrVal: "H"},
						{Kind: OperandNumber, NumVal: -50},
						{Kind: OperandString, StrVal: "ello"},
					},
				}}},
				{Op: "ET", Operands: nil},
			},
			want: "Hello",
		},
		{
			name: "TJ array with word spacing",
			ops: []ContentOperator{
				{Op: "BT", Operands: nil},
				{Op: "TJ", Operands: []ContentOperand{{
					Kind: OperandArray,
					ArrVal: []ContentOperand{
						{Kind: OperandString, StrVal: "Hello"},
						{Kind: OperandNumber, NumVal: -300}, // Large negative spacing = word break
						{Kind: OperandString, StrVal: "World"},
					},
				}}},
				{Op: "ET", Operands: nil},
			},
			want: "Hello World",
		},
		{
			name: "multiple Tj strings",
			ops: []ContentOperator{
				{Op: "BT", Operands: nil},
				{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "First"}}},
				{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "Second"}}},
				{Op: "ET", Operands: nil},
			},
			want: "First Second",
		},
		{
			name: "text outside BT/ET ignored",
			ops: []ContentOperator{
				{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "Outside"}}},
				{Op: "BT", Operands: nil},
				{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "Inside"}}},
				{Op: "ET", Operands: nil},
			},
			want: "Inside",
		},
		{
			name: "line break with T*",
			ops: []ContentOperator{
				{Op: "BT", Operands: nil},
				{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "Line 1"}}},
				{Op: "T*", Operands: nil},
				{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "Line 2"}}},
				{Op: "ET", Operands: nil},
			},
			want: "Line 1\nLine 2",
		},
		{
			name: "line break with quote operator",
			ops: []ContentOperator{
				{Op: "BT", Operands: nil},
				{Op: "Tj", Operands: []ContentOperand{{Kind: OperandString, StrVal: "Line 1"}}},
				{Op: "'", Operands: []ContentOperand{{Kind: OperandString, StrVal: "Line 2"}}},
				{Op: "ET", Operands: nil},
			},
			want: "Line 1\nLine 2",
		},
		{
			name: "empty BT/ET block",
			ops: []ContentOperator{
				{Op: "BT", Operands: nil},
				{Op: "ET", Operands: nil},
			},
			want: "",
		},
		{
			name: "only numbers in TJ array",
			ops: []ContentOperator{
				{Op: "BT", Operands: nil},
				{Op: "TJ", Operands: []ContentOperand{{
					Kind: OperandArray,
					ArrVal: []ContentOperand{
						{Kind: OperandNumber, NumVal: 100},
						{Kind: OperandNumber, NumVal: 200},
					},
				}}},
				{Op: "ET", Operands: nil},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTextFromOperators(tt.ops)
			if got != tt.want {
				t.Errorf("extractTextFromOperators() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractTextQuality performs end-to-end text extraction quality tests
// using actual PDF test files.
func TestExtractTextQuality(t *testing.T) {
	// Test cases using testfixtures for proper path resolution
	// Note: core-multipage has ExpectExtractText: false in testfixtures but actually works
	testCases := []struct {
		name        string
		sampleKey   string
		mustContain []string
	}{
		{
			name:        "core-multipage",
			sampleKey:   "core-multipage",
			mustContain: []string{"Sample PDF"},
		},
	}

	svc := NewService()
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sample, ok := testfixtures.PDFSampleByKey(tc.sampleKey)
			if !ok {
				t.Fatalf("Sample %q not found in testfixtures", tc.sampleKey)
			}

			ref := doc.DocumentRef{Format: doc.FormatPDF, Path: sample.Path()}
			d, err := svc.Open(ctx, ref)
			if err != nil {
				t.Fatalf("Failed to open PDF: %v", err)
			}
			defer d.Close()

			result, err := svc.ExtractText(ctx, d)
			if err != nil {
				t.Fatalf("Failed to extract text: %v", err)
			}

			// Check that text is readable (not individual characters with spaces)
			// The old naive implementation would produce "S a m pl e" instead of "Sample"
			for _, want := range tc.mustContain {
				if !strings.Contains(result.Text, want) {
					previewLen := minInt(len(result.Text), 200)
					t.Errorf("Extracted text does not contain %q.\nGot: %s", want, result.Text[:previewLen])
				}
			}

			// Verify text doesn't have excessive character-by-character spacing
			// (more than 2 spaces between non-whitespace chars indicates poor extraction)
			if strings.Contains(result.Text, "S a m p l e") || strings.Contains(result.Text, "S  a  m  p  l  e") {
				t.Errorf("Text extraction shows character-by-character spacing issues")
			}
		})
	}
}

// Helper function for min (renamed to avoid conflict with Go 1.21+ built-in min)
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
