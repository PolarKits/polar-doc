package doc

import (
	"testing"
)

// TestDocumentRef_Validation tests DocumentRef struct construction and field access
// for both PDF and OFD document references.
func TestDocumentRef_Validation(t *testing.T) {
	tests := []struct {
		name   string
		ref    DocumentRef
		wantFormat Format
		wantPath   string
	}{
		{
			name: "valid PDF document reference",
			ref: DocumentRef{
				Format: FormatPDF,
				Path:   "/path/to/document.pdf",
			},
			wantFormat: FormatPDF,
			wantPath:   "/path/to/document.pdf",
		},
		{
			name: "valid OFD document reference",
			ref: DocumentRef{
				Format: FormatOFD,
				Path:   "/path/to/document.ofd",
			},
			wantFormat: FormatOFD,
			wantPath:   "/path/to/document.ofd",
		},
		{
			name: "document reference with empty path",
			ref: DocumentRef{
				Format: FormatPDF,
				Path:   "",
			},
			wantFormat: FormatPDF,
			wantPath:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ref.Format != tt.wantFormat {
				t.Errorf("Format = %q, want %q", tt.ref.Format, tt.wantFormat)
			}
			if tt.ref.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", tt.ref.Path, tt.wantPath)
			}
		})
	}
}

// TestInfoResult_Fields tests InfoResult struct construction with all fields populated
// and verifies field access.
func TestInfoResult_Fields(t *testing.T) {
	info := InfoResult{
		Format:          FormatPDF,
		Path:            "/docs/test.pdf",
		SizeBytes:       1024,
		DeclaredVersion: "1.4",
		PageCount:       10,
		FileIdentifiers: []string{"id1", "id2"},
		Title:           "Test Document",
		Author:          "Test Author",
		Creator:         "Test Creator",
		Producer:        "Test Producer",
	}

	// Verify all fields are correctly assigned.
	if info.Format != FormatPDF {
		t.Errorf("Format = %q, want %q", info.Format, FormatPDF)
	}
	if info.Path != "/docs/test.pdf" {
		t.Errorf("Path = %q, want %q", info.Path, "/docs/test.pdf")
	}
	if info.SizeBytes != 1024 {
		t.Errorf("SizeBytes = %d, want %d", info.SizeBytes, 1024)
	}
	if info.DeclaredVersion != "1.4" {
		t.Errorf("DeclaredVersion = %q, want %q", info.DeclaredVersion, "1.4")
	}
	if info.PageCount != 10 {
		t.Errorf("PageCount = %d, want %d", info.PageCount, 10)
	}
	if len(info.FileIdentifiers) != 2 {
		t.Errorf("len(FileIdentifiers) = %d, want %d", len(info.FileIdentifiers), 2)
	}
	if info.Title != "Test Document" {
		t.Errorf("Title = %q, want %q", info.Title, "Test Document")
	}
	if info.Author != "Test Author" {
		t.Errorf("Author = %q, want %q", info.Author, "Test Author")
	}
	if info.Creator != "Test Creator" {
		t.Errorf("Creator = %q, want %q", info.Creator, "Test Creator")
	}
	if info.Producer != "Test Producer" {
		t.Errorf("Producer = %q, want %q", info.Producer, "Test Producer")
	}
}

// TestValidationReport_Structure tests ValidationReport construction for both
// valid and invalid states, including Errors slice behavior.
func TestValidationReport_Structure(t *testing.T) {
	tests := []struct {
		name       string
		report     ValidationReport
		wantValid  bool
		wantErrors int
	}{
		{
			name:       "valid document with no errors",
			report:     ValidationReport{Valid: true, Errors: nil},
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name:       "valid document with empty errors slice",
			report:     ValidationReport{Valid: true, Errors: []string{}},
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name:       "invalid document with single error",
			report:     ValidationReport{Valid: false, Errors: []string{"header missing"}},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name:       "invalid document with multiple errors",
			report:     ValidationReport{Valid: false, Errors: []string{"error1", "error2", "error3"}},
			wantValid:  false,
			wantErrors: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.report.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", tt.report.Valid, tt.wantValid)
			}
			if len(tt.report.Errors) != tt.wantErrors {
				t.Errorf("len(Errors) = %d, want %d", len(tt.report.Errors), tt.wantErrors)
			}
		})
	}
}

// TestFormat_Constants tests that Format constants have expected values.
func TestFormat_Constants(t *testing.T) {
	if FormatPDF != "pdf" {
		t.Errorf("FormatPDF = %q, want %q", FormatPDF, "pdf")
	}
	if FormatOFD != "ofd" {
		t.Errorf("FormatOFD = %q, want %q", FormatOFD, "ofd")
	}
}

// TestRefInfo_Structure tests RefInfo struct fields.
func TestRefInfo_Structure(t *testing.T) {
	tests := []struct {
		name         string
		ref          RefInfo
		wantObjNum   int64
		wantGenNum   int64
	}{
		{
			name:       "typical indirect reference",
			ref:        RefInfo{ObjNum: 12, GenNum: 0},
			wantObjNum: 12,
			wantGenNum: 0,
		},
		{
			name:       "reference with non-zero generation",
			ref:        RefInfo{ObjNum: 5, GenNum: 1},
			wantObjNum: 5,
			wantGenNum: 1,
		},
		{
			name:       "zero reference",
			ref:        RefInfo{ObjNum: 0, GenNum: 0},
			wantObjNum: 0,
			wantGenNum: 0,
		},
		{
			name:       "large object number",
			ref:        RefInfo{ObjNum: 999999, GenNum: 65535},
			wantObjNum: 999999,
			wantGenNum: 65535,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ref.ObjNum != tt.wantObjNum {
				t.Errorf("ObjNum = %d, want %d", tt.ref.ObjNum, tt.wantObjNum)
			}
			if tt.ref.GenNum != tt.wantGenNum {
				t.Errorf("GenNum = %d, want %d", tt.ref.GenNum, tt.wantGenNum)
			}
		})
	}
}

// TestFirstPageInfoResult_Structure tests FirstPageInfoResult struct fields.
func TestFirstPageInfoResult_Structure(t *testing.T) {
	rotate90 := int64(90)

	result := FirstPageInfoResult{
		Path:      "/docs/test.pdf",
		PagesRef:  RefInfo{ObjNum: 3, GenNum: 0},
		PageRef:   RefInfo{ObjNum: 4, GenNum: 0},
		Parent:    RefInfo{ObjNum: 3, GenNum: 0},
		MediaBox:  []float64{0, 0, 612, 792},
		Resources: RefInfo{ObjNum: 5, GenNum: 0},
		Contents: []RefInfo{
			{ObjNum: 6, GenNum: 0},
			{ObjNum: 7, GenNum: 0},
		},
		Rotate: &rotate90,
	}

	// Verify all fields are correctly assigned.
	if result.Path != "/docs/test.pdf" {
		t.Errorf("Path = %q, want %q", result.Path, "/docs/test.pdf")
	}
	if result.PagesRef.ObjNum != 3 {
		t.Errorf("PagesRef.ObjNum = %d, want %d", result.PagesRef.ObjNum, 3)
	}
	if result.PageRef.ObjNum != 4 {
		t.Errorf("PageRef.ObjNum = %d, want %d", result.PageRef.ObjNum, 4)
	}
	if result.Parent.ObjNum != 3 {
		t.Errorf("Parent.ObjNum = %d, want %d", result.Parent.ObjNum, 3)
	}
	if len(result.MediaBox) != 4 {
		t.Errorf("len(MediaBox) = %d, want %d", len(result.MediaBox), 4)
	}
	if result.Resources.ObjNum != 5 {
		t.Errorf("Resources.ObjNum = %d, want %d", result.Resources.ObjNum, 5)
	}
	if len(result.Contents) != 2 {
		t.Errorf("len(Contents) = %d, want %d", len(result.Contents), 2)
	}
	if result.Rotate == nil || *result.Rotate != 90 {
		t.Errorf("Rotate = %v, want 90", result.Rotate)
	}
}

// TestFirstPageInfoResult_NilRotate tests FirstPageInfoResult with nil Rotate field.
func TestFirstPageInfoResult_NilRotate(t *testing.T) {
	result := FirstPageInfoResult{
		Path:     "/docs/test.pdf",
		PagesRef: RefInfo{ObjNum: 3, GenNum: 0},
		PageRef:  RefInfo{ObjNum: 4, GenNum: 0},
		MediaBox: []float64{0, 0, 612, 792},
		// Rotate is intentionally nil (no rotation specified).
		Rotate: nil,
	}

	if result.Rotate != nil {
		t.Errorf("Rotate = %v, want nil", *result.Rotate)
	}
}

// TestInfoResult_MinimalFields tests InfoResult with only required fields populated.
func TestInfoResult_MinimalFields(t *testing.T) {
	info := InfoResult{
		Format:    FormatOFD,
		Path:      "/docs/minimal.ofd",
		SizeBytes: 512,
		// Leave optional fields at zero values.
		PageCount:       0,
		DeclaredVersion: "",
		Title:           "",
		Author:          "",
		Creator:         "",
		Producer:        "",
		FileIdentifiers: nil,
	}

	if info.Format != FormatOFD {
		t.Errorf("Format = %q, want %q", info.Format, FormatOFD)
	}
	if info.Path != "/docs/minimal.ofd" {
		t.Errorf("Path = %q, want %q", info.Path, "/docs/minimal.ofd")
	}
	if info.SizeBytes != 512 {
		t.Errorf("SizeBytes = %d, want %d", info.SizeBytes, 512)
	}
	if info.PageCount != 0 {
		t.Errorf("PageCount = %d, want %d (zero value)", info.PageCount, 0)
	}
	if info.DeclaredVersion != "" {
		t.Errorf("DeclaredVersion = %q, want empty string", info.DeclaredVersion)
	}
}
