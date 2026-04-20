package pdf

import "fmt"

// PDFVersion represents a PDF specification version as major and minor integers.
// Use the predefined constants (PDF14, PDF17, PDF20, etc.) rather than constructing
// versions manually, to avoid accidental mismatches in comparison logic.
type PDFVersion struct {
	Major int
	Minor int
}

// Predefined PDF version constants covering the range relevant to PolarDoc.
var (
	PDF10 = PDFVersion{1, 0}
	PDF13 = PDFVersion{1, 3}
	PDF14 = PDFVersion{1, 4}
	PDF15 = PDFVersion{1, 5}
	PDF16 = PDFVersion{1, 6}
	PDF17 = PDFVersion{1, 7}
	PDF20 = PDFVersion{2, 0}
)

// IsZero reports whether v is the zero value (unset).
func (v PDFVersion) IsZero() bool { return v.Major == 0 && v.Minor == 0 }

// AtLeast reports whether v is greater than or equal to other.
func (v PDFVersion) AtLeast(other PDFVersion) bool {
	if v.Major != other.Major {
		return v.Major > other.Major
	}
	return v.Minor >= other.Minor
}

// String returns the canonical version string as it appears in the PDF header (e.g. "1.7").
func (v PDFVersion) String() string {
	if v.IsZero() {
		return ""
	}
	return string(rune('0'+v.Major)) + "." + string(rune('0'+v.Minor))
}

// parsePDFVersion parses a version string of the form "X.Y" as read from a %PDF-X.Y header.
// Returns the zero PDFVersion and an error if the string is not a valid two-part version.
func parsePDFVersion(s string) (PDFVersion, error) {
	if len(s) < 3 || s[1] != '.' {
		return PDFVersion{}, fmt.Errorf("invalid PDF version %q", s)
	}
	major := int(s[0] - '0')
	minor := int(s[2] - '0')
	if major < 1 || major > 9 || minor < 0 || minor > 9 {
		return PDFVersion{}, fmt.Errorf("PDF version out of range: %q", s)
	}
	return PDFVersion{major, minor}, nil
}

// EncryptionAlgorithm identifies the encryption algorithm declared in a PDF /Encrypt dictionary.
type EncryptionAlgorithm int

const (
	// EncryptNone indicates no encryption is present.
	EncryptNone EncryptionAlgorithm = iota
	// EncryptRC4_40 is the 40-bit RC4 algorithm (PDF 1.1–1.3, deprecated).
	EncryptRC4_40
	// EncryptRC4_128 is the 128-bit RC4 algorithm (PDF 1.4–1.6, deprecated).
	EncryptRC4_128
	// EncryptAES_128 is the 128-bit AES algorithm (PDF 1.6).
	EncryptAES_128
	// EncryptAES_256 is the 256-bit AES algorithm (PDF 1.7 extension, PDF 2.0 standard).
	EncryptAES_256
	// EncryptUnknown indicates an encryption entry was found but the algorithm could not be identified.
	EncryptUnknown
)

// xrefKind distinguishes how an object's location is stored in the xref index.
type xrefKind int

const (
	// xrefKindFree marks a free (deleted) object slot.
	xrefKindFree xrefKind = iota
	// xrefKindDirect stores the object at a direct file byte offset.
	xrefKindDirect
	// xrefKindInObjStm stores the object inside a compressed object stream (ObjStm, PDF 1.5+).
	xrefKindInObjStm
)

// xrefLocation describes where a specific PDF object resides in the file.
// It abstracts the difference between traditional xref table entries and
// cross-reference stream entries (ISO 32000-1 §7.5.8).
type xrefLocation struct {
	Kind       xrefKind
	Offset     int64
	ObjStmNum  int64
	IndexInStm int
	Generation int
}

// PDFFeatureSet describes the structural features detected in an opened PDF document.
// These fields are probed from the document itself and must not be inferred from
// DeclaredVersion alone — a document may claim version 1.4 but contain 1.5 structures.
type PDFFeatureSet struct {
	// DeclaredVersion is the version read from the %PDF-X.Y header comment.
	DeclaredVersion PDFVersion
	// EffectiveVersion is the maximum of DeclaredVersion and the version implied
	// by detected features (e.g. xref streams imply at least PDF 1.5).
	EffectiveVersion PDFVersion

	// xref structure
	HasTraditionalXRef bool // traditional xref table (PDF 1.0+)
	HasXRefStream       bool // cross-reference stream (PDF 1.5+)
	// IsHybridXRef is true when both xref table and xref stream are present.
	// This is a known generator bug. The xref stream takes precedence (ISO 32000-1 §C.2).
	IsHybridXRef bool

	// HasObjectStreams is true when at least one ObjStm entry was found in the xref index.
	HasObjectStreams bool // object streams (PDF 1.5+)

	// document structure
	HasIncrementalUpdates bool // more than one xref/trailer revision present
	IsLinearized         bool // linearized (fast web view) layout
	IsEncrypted           bool

	// EncryptionAlgorithm describes the algorithm in the /Encrypt dictionary.
	// Valid only when IsEncrypted is true.
	EncryptionAlgorithm EncryptionAlgorithm

	// metadata locations
	HasInfoDict  bool // traditional /Info dictionary (PDF 1.0–1.7)
	HasXMPStream bool // XMP metadata stream (PDF 1.4+; primary in PDF 2.0)
}

// CompatFix is a bitmask of known spec ambiguities and generator bugs that
// CompatReader handles silently instead of returning an error.
// Each set bit enables one compensating behavior and records a CompatWarning.
type CompatFix uint64

const (
	// FixHybridXRef: when both xref table and xref stream are present, prefer the
	// xref stream as the authoritative index (ISO 32000-1 §C.2).
	FixHybridXRef CompatFix = 1 << iota

	// FixBrokenStartxref: if the startxref offset points to invalid content, scan
	// backward from EOF to locate the last valid xref or obj marker.
	FixBrokenStartxref

	// FixMissingEOF: tolerate a missing %%EOF marker at the end of the file.
	FixMissingEOF

	// FixInfoDictUTF16NoBOM: treat Info dict strings that have no BOM but appear to
	// be UTF-16BE as UTF-16BE. Produced by Acrobat prior to version 6.0.
	FixInfoDictUTF16NoBOM

	// FixStreamLengthMismatch: when the /Length value in a stream dictionary does
	// not match the actual stream boundary, use the "endstream" keyword as the
	// authoritative delimiter.
	FixStreamLengthMismatch

	// FixTrailerPrevChain: stop following the /Prev chain if it points to an already-
	// visited offset or to an invalid location, preventing infinite loops.
	FixTrailerPrevChain

	// FixNullObjectRef: treat indirect references of the form "0 0 R" as null rather
	// than returning an error.
	FixNullObjectRef

	// FixEmptyEncryptDict: if the /Encrypt entry exists but its dictionary is empty,
	// treat the document as unencrypted.
	FixEmptyEncryptDict
)

// DefaultCompatFixes is the recommended fix set covering the most common real-world
// generator defects catalogued by the PDF Association issue tracker.
var DefaultCompatFixes = FixHybridXRef |
	FixBrokenStartxref |
	FixMissingEOF |
	FixInfoDictUTF16NoBOM |
	FixStreamLengthMismatch |
	FixTrailerPrevChain |
	FixNullObjectRef |
	FixEmptyEncryptDict

// CompatWarning records a single silent fix event. Callers may surface these
// warnings to users or logging systems.
type CompatWarning struct {
	// Fix identifies which CompatFix was triggered.
	Fix CompatFix
	// Detail is a human-readable description of the specific deviation encountered.
	Detail string
}
