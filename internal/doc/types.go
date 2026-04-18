package doc

// Format identifies a document format domain.
//
// These are routing identifiers, not standard version identifiers.
// FormatPDF maps to ISO 32000-2:2020 (PDF 2.0) as the normative reference,
// but this type only carries the format name, not the specific version.
type Format string

const (
	// FormatPDF identifies PDF documents (ISO 32000-2:2020 baseline).
	FormatPDF Format = "pdf"
	// FormatOFD identifies OFD documents (GB/T 33190-2016 baseline).
	FormatOFD Format = "ofd"
)

// DocumentRef is a lightweight handle that points to a document input.
//
// It intentionally carries only routing and identity metadata.
// It is not a shared in-memory document model.
//
// This type does not represent a fully parsed document structure.
// It is analogous to a file path + format tag, not a document object graph.
type DocumentRef struct {
	Format Format
	Path   string
}

// InfoResult is minimal document metadata returned by the info command.
//
// This type carries only a narrow, stability-friendly subset of properties.
// It does not correspond to any specific standard's "Info dictionary" or metadata
// object; it is a phase-1 transport struct.
//
// DeclaredVersion for PDF reflects the %PDF-X.Y header comment. For OFD it is
// currently empty (phase-1 stub).
type InfoResult struct {
	Format          Format
	Path            string
	SizeBytes       int64
	DeclaredVersion string
}

// ValidationReport is a structured validation output.
//
// Valid=false indicates the document failed a basic structural check.
// Errors are human-readable strings derived from format-specific rules;
// they are NOT a complete enumeration of all possible standard violations.
//
// Phase-1 coverage for PDF: only the header presence rule from ISO 32000-2.
// Phase-1 coverage for OFD: only the package entry presence rules from GB/T 33190-2016.
type ValidationReport struct {
	Valid  bool
	Errors []string
}

// PreviewRequest describes a requested preview rendering.
//
// Page and DPI are hints. Format-specific implementations may interpret these
// differently or ignore them if unsupported.
type PreviewRequest struct {
	Page int
	DPI  int
}

// PreviewResult describes produced preview metadata.
//
// MediaType is a MIME type string (e.g. "image/png"). Data is the raw payload.
// Phase-1: this is a stub; preview rendering is not implemented.
type PreviewResult struct {
	MediaType string
	Data      []byte
}

// TextResult describes extracted text output.
//
// Text is returned as a single concatenated string. The extraction rules,
// ordering guarantees, and content completeness are format-defined.
// Phase-1: both PDF and OFD return empty string (stub).
//
// # Version Upgrade Note
//
// Upgrading a document from an older version to a newer version (e.g. PDF 1.4 → PDF 2.0)
// is a planned capability. It requires a writer pipeline that does not yet exist.
// The current TextExtractor interface does not distinguish input version from output version.
type TextResult struct {
	Text string
}

// SignRequest describes a signing request at the capability layer.
//
// Profile is format-specific. For PDF, profiles map to ISO 32000-2 approval
// and certification signatures. For OFD, profiles map to GB/T 33190-2016
// digital signature rules.
//
// Phase-1 coverage: signing is not implemented.
type SignRequest struct {
	Profile string
	Reason  string
}

// SignResult describes signature metadata returned after signing.
//
// Method describes the cryptographic algorithm used. Signature is the raw
// signature bytes. This struct does not include certificate chain, timestamp,
// or revocation information; those are future extensions.
//
// Phase-1 coverage: signing is not implemented.
type SignResult struct {
	Method    string
	Signature []byte
}

// FirstPageInfoResult holds first page structure information.
//
// This is a format-neutral result type. PDF implementations populate
// fields from PDF primitives; OFD does not support this capability.
// Fields use plain Go types to avoid leaking format-specific internals
// to the command layer.
type FirstPageInfoResult struct {
	Path      string
	PagesRef  RefInfo
	PageRef   RefInfo
	Parent    RefInfo
	MediaBox  []float64
	Resources RefInfo
	Contents  []RefInfo
	Rotate    *int64
}

// RefInfo is a minimal indirect reference representation.
type RefInfo struct {
	ObjNum int64
	GenNum int64
}
