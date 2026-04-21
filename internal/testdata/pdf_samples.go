package testdata

import (
	"path/filepath"
	"runtime"
)

// PDFSampleIntegrity indicates the integrity status of a PDF sample fixture.
type PDFSampleIntegrity string

const (
	// PDFSampleIntegrityValid indicates the PDF is structurally valid.
	PDFSampleIntegrityValid PDFSampleIntegrity = "valid_pdf"
	// PDFSampleIntegrityCorrupted indicates the PDF is intentionally corrupted for error testing.
	PDFSampleIntegrityCorrupted PDFSampleIntegrity = "corrupted_pdf"
	// PDFSampleIntegrityPlaceholder indicates the file is not a real PDF (reserved for future use).
	PDFSampleIntegrityPlaceholder PDFSampleIntegrity = "placeholder_non_pdf"
)

// PDFSample describes a single PDF fixture file and its test expectations.
type PDFSample struct {
	// Key is the unique identifier for this sample (used in tests and lookups).
	Key string
	// Filename is the actual file name in testdata/pdf/.
	Filename string
	// Category groups samples by purpose (core, feature, standard, version-compat, etc.).
	Category string
	// Description explains what this sample is meant to test.
	Description string
	// DeclaredVersionHint is the PDF version declared in the file header (for reference, not validated).
	DeclaredVersionHint string
	// Integrity indicates whether this sample is valid, corrupted, or a placeholder.
	Integrity PDFSampleIntegrity
	// ExpectFirstPageInfo is true when the fixture is expected to support first-page info extraction.
	ExpectFirstPageInfo bool
	// ExpectExtractText is true when the fixture is expected to yield non-empty extracted text.
	ExpectExtractText bool
	// ExpectFileIDs is true when the fixture is expected to have file identifiers in its trailer.
	ExpectFileIDs bool
}

// Path returns the absolute path to the PDF fixture file.
func (s PDFSample) Path() string {
	return filepath.Join(repoRoot(), "testdata", "pdf", s.Filename)
}

// PDFSamples returns all registered PDF fixtures.
func PDFSamples() []PDFSample {
	return append([]PDFSample(nil), pdfSamples...)
}

// PDFSampleByKey returns the PDF fixture with the given key, or (zero, false) if not found.
func PDFSampleByKey(key string) (PDFSample, bool) {
	for _, sample := range pdfSamples {
		if sample.Key == key {
			return sample, true
		}
	}
	return PDFSample{}, false
}

func repoRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(filepath.Dir(filename)))
}

var pdfSamples = []PDFSample{
	{Key: "core-minimal", Filename: "test_core_minimal_v1.5.pdf", Category: "core", Description: "Minimal PDF 1.5 fixture", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "core-latex-standard", Filename: "test_core_latex_standard_v1.5.pdf", Category: "core", Description: "LaTeX-generated baseline PDF 1.5 fixture", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: true, ExpectFileIDs: true},
	{Key: "core-multipage", Filename: "test_core_multipage_v1.3.pdf", Category: "core", Description: "Multi-page PDF 1.3 fixture", DeclaredVersionHint: "1.3", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "core-multicolumn", Filename: "test_core_multicolumn_v1.5.pdf", Category: "core", Description: "Multi-column PDF 1.5 fixture", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: true, ExpectFileIDs: true},
	{Key: "feature-acroform", Filename: "test_feat_acroform_v1.5.pdf", Category: "feature", Description: "Interactive form fixture", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "feature-attachment", Filename: "test_feat_attachment_v1.5.pdf", Category: "feature", Description: "Embedded attachment fixture", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "feature-cmyk", Filename: "test_feat_cmyk_v1.4.pdf", Category: "feature", Description: "CMYK color-space fixture", DeclaredVersionHint: "1.4", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "feature-complex-attachments", Filename: "test_feat_complex_attachments_v1.5.pdf", Category: "feature", Description: "Complex attachment structure fixture", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "feature-encrypted", Filename: "test_feat_encrypted_v1.5.pdf", Category: "feature", Description: "Encrypted PDF fixture", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "feature-fillable", Filename: "test_feat_fillable_v1.6.pdf", Category: "feature", Description: "Fillable form fixture", DeclaredVersionHint: "1.6", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "feature-image-mask", Filename: "test_feat_image_mask.pdf", Category: "feature", Description: "Image mask fixture", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "feature-links", Filename: "test_feat_links_v1.5.pdf", Category: "feature", Description: "Links and annotations fixture", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "feature-rtl-arabic", Filename: "test_feat_rtl_arabic_v1.5.pdf", Category: "feature", Description: "Right-to-left Arabic text fixture", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "feature-tagged", Filename: "test_feat_tagged_v1.7.pdf", Category: "feature", Description: "Tagged PDF fixture", DeclaredVersionHint: "1.7", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "feature-transparency", Filename: "test_feat_transparency_v1.4.pdf", Category: "feature", Description: "Transparency fixture", DeclaredVersionHint: "1.4", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "performance-large-doc", Filename: "test_perf_large_doc_v1.4.pdf", Category: "performance", Description: "Large-document fixture", DeclaredVersionHint: "1.4", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "standard-pdf20-basic", Filename: "test_std_pdf20_basic.pdf", Category: "standard", Description: "Basic PDF 2.0 fixture", DeclaredVersionHint: "2.0", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "standard-pdf20-image-bpc", Filename: "test_std_pdf20_image_bpc.pdf", Category: "standard", Description: "PDF 2.0 image bits-per-component fixture", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "standard-pdf20-incremental", Filename: "test_std_pdf20_incremental.pdf", Category: "standard", Description: "PDF 2.0 incremental update fixture", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "standard-pdf20-output-intent", Filename: "test_std_pdf20_output_intent.pdf", Category: "standard", Description: "PDF 2.0 output intent fixture", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "standard-pdf20-utf8-annotation", Filename: "test_std_pdf20_utf8_annotation.pdf", Category: "standard", Description: "PDF 2.0 UTF-8 annotation fixture", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "standard-pdf20-utf8", Filename: "test_std_pdf20_utf8_v2.0.pdf", Category: "standard", Description: "PDF 2.0 UTF-8 and tagged-structure fixture", DeclaredVersionHint: "2.0", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: true, ExpectFileIDs: false},
	{Key: "standard-pdf20", Filename: "test_std_pdf20_v2.0.pdf", Category: "standard", Description: "General PDF 2.0 fixture", DeclaredVersionHint: "2.0", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "standard-pdfa2b", Filename: "test_std_pdfa2b_v1.7.pdf", Category: "standard", Description: "PDF/A-2b fixture", DeclaredVersionHint: "1.7", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "standard-pdfa-archival", Filename: "test_std_pdfa_archival_v1.4.pdf", Category: "standard", Description: "PDF/A archival fixture", DeclaredVersionHint: "1.4", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "version-compat-v1.0", Filename: "test_ver_compat_v1.0.pdf", Category: "version-compat", Description: "Version compatibility fixture for PDF 1.0", DeclaredVersionHint: "1.0", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "version-compat-v1.4", Filename: "test_ver_compat_v1.4.pdf", Category: "version-compat", Description: "Version compatibility fixture for PDF 1.4", DeclaredVersionHint: "1.4", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: true, ExpectFileIDs: true},
	{Key: "version-compat-v1.7", Filename: "test_ver_compat_v1.7.pdf", Category: "version-compat", Description: "Known-corrupted version compatibility fixture for PDF 1.7", DeclaredVersionHint: "1.7", Integrity: PDFSampleIntegrityCorrupted, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "error-corrupted", Filename: "test_err_corrupted.pdf", Category: "error", Description: "Corrupted PDF fixture used to validate error paths", DeclaredVersionHint: "1.5", Integrity: PDFSampleIntegrityCorrupted, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "version-v1.0", Filename: "test_ver_v1.0.pdf", Category: "version", Description: "Legacy version fixture named v1.0", DeclaredVersionHint: "1.0", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "version-v1.1", Filename: "test_ver_v1.1.pdf", Category: "version", Description: "Legacy version fixture named v1.1", DeclaredVersionHint: "1.1", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: true},
	{Key: "legacy-pdf-ua-sample", Filename: "pdf-ua-sample.pdf", Category: "legacy", Description: "Legacy PDF/UA sample", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "legacy-pdf20-attachment", Filename: "pdf20-with-attachment.pdf", Category: "legacy", Description: "Legacy PDF 2.0 attachment sample", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: true, ExpectExtractText: false, ExpectFileIDs: false},
	{Key: "legacy-with-table", Filename: "with-table.pdf", Category: "legacy", Description: "Legacy table sample", Integrity: PDFSampleIntegrityValid, ExpectFirstPageInfo: false, ExpectExtractText: false, ExpectFileIDs: false},
}
