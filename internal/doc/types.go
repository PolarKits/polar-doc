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
	// Format is the document format domain (PDF or OFD).
	Format Format
	// Path is the file system path to the document.
	Path string
}

// InfoResult is minimal document metadata returned by the info command.
//
// This type carries only a narrow, stability-friendly subset of properties.
// It does not correspond to any specific standard's "Info dictionary" or metadata
// object; it is a phase-1 transport struct.
//
// Field population by format:
//   - Format, Path, SizeBytes: populated for both PDF and OFD
//   - DeclaredVersion: PDF reads %PDF-X.Y header; OFD reads Version attribute from OFD.xml root element
//   - PageCount: PDF populated from /Count in root /Pages dict; OFD from Document.xml <Page> count
//   - FileIdentifiers: PDF populates from trailer /ID array (Phase-1); OFD is unused (no FileIdentifiers equivalent in GB/T 33190-2016)
//   - Title, Author, Creator, Producer: PDF populates from InfoDict; OFD does not
//
// Empty string on optional fields means the metadata is not available.
// Zero PageCount means page count is unknown or not yet implemented.
type InfoResult struct {
	// Format is the document format domain (PDF or OFD).
	Format Format
	// Path is the file system path to the document.
	Path string
	// SizeBytes is the file size in bytes.
	SizeBytes int64
	// DeclaredVersion is the format version declared in the document header.
	// PDF: from %PDF-X.Y header; OFD: from Version attribute in OFD.xml.
	DeclaredVersion string

	// PageCount holds the document page count.
	// PDF: populated from /Count in the root /Pages dictionary. OFD: populated from Document.xml.
	PageCount int

	// FileIdentifiers: PDF populates from trailer /ID array; OFD is unused (no FileIdentifiers equivalent in GB/T 33190-2016).
	// Empty slice means no file identifiers are available.
	FileIdentifiers []string

	// Title: PDF populates from InfoDict /Title; OFD does not populate.
	// Empty string means title is not available.
	Title string

	// Author: PDF populates from InfoDict /Author; OFD does not populate.
	// Empty string means author is not available.
	Author string

	// Creator: PDF populates from InfoDict /Creator; OFD does not populate.
	// Empty string means creator is not available.
	Creator string

	// Producer: PDF populates from InfoDict /Producer; OFD does not populate.
	// Empty string means producer is not available.
	Producer string

	// Seals holds electronic seal metadata for OFD documents.
	// nil if the document has no electronic seals or is not OFD.
	// For OFD: populated from parsing Signatures.xml and associated Seal.esl files.
	Seals []SealSummary

	// Fonts holds font resource metadata for OFD documents.
	// nil if the document has no fonts or is not OFD.
	// For OFD: populated from parsing PublicRes.xml and DocumentRes.xml.
	Fonts []FontSummary

	// MediaFiles holds multimedia resource metadata for OFD documents.
	// nil if the document has no multimedia or is not OFD.
	// For OFD: populated from parsing PublicRes.xml and DocumentRes.xml.
	MediaFiles []MediaSummary
}

// SealSummary holds basic electronic seal metadata from an OFD document.
// This is a phase-1 transport struct; cryptographic verification is not performed.
type SealSummary struct {
	// ID is the signature/seal identifier from the OFD package.
	ID int64
	// Version is the seal format version (e.g. "1.0").
	Version string
	// Width and Height describe the seal picture dimensions in pixels (0 if unknown).
	Width  int64
	Height int64
	// PictureFormat is the format of the embedded seal image (e.g. "PNG", empty if none).
	PictureFormat string
}

// FontSummary holds basic font metadata from an OFD document.
type FontSummary struct {
	// FontID is the resource identifier from the OFD package.
	FontID int64
	// FamilyName is the font family name (e.g. "SimHei", "Arial").
	FamilyName string
	// FontName is the specific font name within the family.
	FontName string
}

// MediaSummary holds basic multimedia metadata from an OFD document.
type MediaSummary struct {
	// MediaID is the resource identifier from the OFD package.
	MediaID int64
	// MediaType describes the type of multimedia (e.g. "Image", "Audio").
	MediaType string
	// Format is the file format of the media (e.g. "PNG", "JPEG", "MP3").
	Format string
}

// ValidationReport is a structured validation output.
//
// Valid=false indicates the document failed a basic structural check.
// Errors are human-readable strings derived from format-specific rules;
// they are NOT a complete enumeration of all possible standard violations.
// Warnings contains non-fatal issues that don't invalidate the document but may indicate problems.
//
// Phase-1 coverage for PDF: header presence check (%PDF- prefix format) per ISO 32000-2.
// Phase-1 coverage for OFD: package entry presence and DocRoot integrity per GB/T 33190-2016.
//
// All fields are structural checks only. Semantic validation (e.g., font licensing,
// accessibility requirements, digital signature validity) is not performed in phase-1.
type ValidationReport struct {
	// Valid is true when the document passes basic structural checks for its format.
	// A document may be structurally valid yet semantically non-compliant (Phase-2 scope).
	Valid bool
	// Errors contains human-readable structural failure reasons.
	// This is not an exhaustive list of standard violations.
	Errors []string
	// Warnings contains non-fatal issues that don't invalidate the document.
	// Examples: unknown encryption algorithm, deprecated structures.
	Warnings []string
}

// PreviewRequest describes a requested preview rendering.
//
// Page and DPI are hints. Format-specific implementations may interpret these
// differently or ignore them if unsupported.
type PreviewRequest struct {
	// Page is the requested page number (1-indexed). Zero means no specific page requested.
	Page int
	// DPI is the requested resolution in dots per inch. Zero means use format default.
	DPI int
}

// PreviewResult describes produced preview metadata.
//
// MediaType is a MIME type string (e.g. "image/png"). Data is the raw payload.
// Phase-1: this is a stub; preview rendering is not implemented.
type PreviewResult struct {
	// MediaType is the MIME type of the preview data (e.g. "image/png", "image/jpeg").
	MediaType string
	// Data is the raw preview payload bytes.
	Data []byte
}

// TextResult describes extracted text output.
//
// Text is returned as a single concatenated string. The extraction rules,
// ordering guarantees, and content completeness are format-defined.
// Phase-1:
//   - PDF may return partial extracted text for supported content streams.
//   - OFD returns text extracted from TextCode elements across all pages.
//
// # Version Upgrade Note
//
// Upgrading a document from an older version to a newer version (e.g. PDF 1.4 → PDF 2.0)
// is a planned capability. It requires a writer pipeline that does not yet exist.
// The current TextExtractor interface does not distinguish input version from output version.
type TextResult struct {
	// Text is the extracted text content. Format-specific extraction rules determine
	// ordering and completeness. Empty string means no text was extracted.
	Text string
}

// SignRequest describes a signing request at the capability layer.
//
// Profile is format-specific. For PDF, profiles map to ISO 32000-2 approval
// and certification signatures. For OFD, profiles map to GB/T 33190-2016
// digital signature rules.
//
// Phase-1 coverage: signing is not implemented. The fields below represent
// capability-layer reservations for Phase-2 signature and timestamp support.
type SignRequest struct {
	// Profile is the format-specific signature profile identifier.
	// For PDF: maps to ISO 32000-2 approval/certification signatures.
	// For OFD: maps to GB/T 33190-2016 digital signature rules.
	Profile string

	// Reason is the human-readable reason for signing.
	Reason string

	// HashAlgorithm reserves the hash algorithm identifier for Phase-2.
	// Examples: "SHA-256", "SHA-384", "SHA-512".
	// Empty string means not specified.
	HashAlgorithm string

	// CertSource reserves the certificate source or identifier for Phase-2.
	// This may be a keystore alias, PKCS#11 slot, or explicit cert fingerprint.
	// Empty string means not specified.
	CertSource string

	// TimestampURL reserves the RFC 3161 timestamp service URL for Phase-2.
	// When non-empty, the signer will request a trusted timestamp.
	// Empty string means no timestamp requested.
	TimestampURL string
}

// SignResult describes signature metadata returned after signing.
//
// Method describes the cryptographic algorithm used. Signature is the raw
// signature bytes. This struct does not include certificate chain, timestamp,
// or revocation information; those are future extensions.
//
// Phase-1 coverage: signing is not implemented. The fields below represent
// capability-layer reservations for Phase-2 signature and timestamp support.
type SignResult struct {
	// Method is the cryptographic algorithm used for signing (e.g. "RSA", "ECDSA").
	Method string
	// Signature is the raw signature bytes.
	Signature []byte

	// CertDigest reserves the SHA-256 digest of the signing certificate for Phase-2.
	// Hex-encoded string. Empty if certificate not available.
	CertDigest string

	// HasTimestamp reserves the timestamp indicator for Phase-2.
	// True if the signature includes an RFC 3161 trusted timestamp.
	HasTimestamp bool

	// TimestampURL reserves the timestamp service URL used for Phase-2, if any.
	// Empty string if no timestamp was requested or obtained.
	TimestampURL string
}

// FirstPageInfoResult holds first page structure information.
//
// The result type is format-neutral (plain Go types) to avoid leaking format
// internals to the command layer.
//
// PDF implementations populate fields from PDF page primitives (page tree,
// page dictionary, content stream references). OFD implementations map
// PhysicalBox from Document.xml's PageArea to MediaBox; other fields remain
// zero-valued since OFD does not have equivalent concepts (no indirect
// reference model, no /Resources or /Contents dictionary chains).
type FirstPageInfoResult struct {
	// Path is the file path of the document.
	Path string
	// PagesRef is the indirect reference to the root Pages object (/Type /Pages).
	PagesRef RefInfo
	// PageRef is the indirect reference to the first page object (/Type /Page).
	PageRef RefInfo
	// Parent is the indirect reference to the parent Pages object containing this page.
	Parent RefInfo
	// MediaBox is the page media box rectangle [llx, lly, urx, ury] in default user space units.
	MediaBox []float64
	// Resources is the indirect reference to the resource dictionary for this page.
	Resources RefInfo
	// Contents is a slice of indirect references to content streams for this page.
	Contents []RefInfo
	// Rotate is the page rotation in degrees (0, 90, 180, 270). Nil means no rotation specified.
	Rotate *int64
}

// RefInfo is a minimal indirect reference representation.
type RefInfo struct {
	// ObjNum is the object number (e.g. the "N" in "N 0 R" indirect reference).
	ObjNum int64
	// GenNum is the generation number (e.g. the "0" in "N 0 R" indirect reference).
	GenNum int64
}
