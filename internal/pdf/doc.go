// Package pdf contains PDF-specific domain logic and operations.
//
// The PDF domain must not depend on the OFD domain.
//
// # Standards Baseline
//
// This package implements PDF semantics mapped to ISO 32000-2:2020 (PDF 2.0).
// The standard baseline is PDF 2.0; older versions (PDF 1.4, 1.7) are supported
// for read-only compatibility at the structural level this package currently covers.
//
// # Version Policy
//
// This section defines the explicit policy for PDF version handling.
//
// ## Read Compatibility
//
// The compatibility target for reading is PDF 2.0, 1.7, and 1.4 files.
// Read-oriented support in phase-1 covers:
//   - Header declared version (%PDF-X.Y comment)
//   - Traditional xref table traversal and XRef stream decoding (including Prev chain)
//   - Trailer /ID array extraction
//   - InfoDict metadata (Title, Author, Creator, Producer)
//   - PageCount from root /Pages /Count
//   - FirstPageInfo: Catalog→Pages→Page traversal with MediaBox/Resources/Contents/Rotate extraction
//   - Full-document text extraction from content streams via content operator parsing
//   - Stream filter support: FlateDecode, ASCIIHexDecode, ASCII85Decode, LZWDecode, RunLengthDecode
//   - Font encoding: WinAnsiEncoding, MacRomanEncoding, StandardEncoding, MacExpertEncoding,
//
// Full semantic compatibility with ISO 32000-2 or ISO 32000-1 is NOT claimed.
// Specifically: ObjStm entries are resolved via resolveFromObjStm but object stream
// compression is read-only; content operator parsing is implemented for BT/ET, Tj/TJ,
// and TJ array spacing; font mapping supports WinAnsi/MacRoman/Standard/MacExpertEncoding
// and ToUnicode CMap (partial); xref/trailer/InfoDict are read but comprehensive
// integrity validation is not performed.
//
// ## Write / Upgrade Policy
//
// Upgrading an input file from an older PDF version to a newer output version
// (e.g. PDF 1.4 → PDF 2.0) is a future capability requiring an explicit
// writer pipeline design. The current implementation does NOT perform implicit
// version upgrades: reading a PDF 1.4 file and outputting it does not silently
// produce a PDF 2.0 file. Any future upgrade output must be an explicit
// opt-in strategy decided at the application or command layer, not a hidden
// behavior in this package.
//
// RewriteFile produces single-revision output only; incremental updates are
// not supported in phase-1.
//
// # Phase-1 Implementation Scope
//
// Implemented (phase-1):
//   - Open: acquires file handle, reads %PDF-X.Y header version
//   - Info: provides DeclaredVersion, SizeBytes, Format, PageCount (from /Pages /Count),
//     FileIdentifiers (from trailer /ID array), Title/Author/Creator/Producer (from InfoDict)
//   - Validate: 5-level structural validation (Header → XRef → Trailer → Catalog → Pages)
//   - FirstPageInfo: traverses Catalog→Pages→Page chain, extracts MediaBox, Resources,
//     Contents, Rotate (with inheritance)
//   - xref traversal: reads traditional xref tables and XRef streams (including Prev chain);
//     Type 2 (ObjStm) entries are resolved via resolveFromObjStm
//   - ValidateDeep: comprehensive integrity validation covering xref table/stream integrity,
//     object accessibility, trailer validity, and cross-reference consistency
//   - ExtractText: full-document text extraction with content operator parsing (BT/ET, Tj/TJ,
//     TJ array spacing); supports WinAnsiEncoding, MacRomanEncoding, ToUnicode CMap
//   - CopyFile: raw byte copy to destination path
//   - RewriteFile: normalizes incremental PDFs to single-revision output; follows full
//     xref chain, writes sequential live objects with fresh byte offsets, emits compact
//     xref table and fresh trailer; ObjStm objects are preserved but not expanded
//   - content_parser.go: content stream operator parsing (text blocks, string operands,
//     hex strings, content operators, TJ array spacing analysis)
//   - stream_filter.go: multi-filter framework (FlateDecode, ASCIIHexDecode, ASCII85Decode,
	//     LZWDecode, RunLengthDecode; supports filter chains)
//   - font_encoding.go: built-in encoding tables (WinAnsiEncoding 128-255, MacRomanEncoding
//     128-255, StandardEncoding, /Differences array parsing, applyByteMapping for
//     font-to-Unicode mapping)
//
// Not implemented in phase-1 (future work):
//   - CIDFont CMap-based font encoding
//   - Incremental update writer pipeline
//   - Preview rendering and thumbnail generation
//   - Digital signature parsing and validation (ISO 32000-2 §12.8)
//   - PDF version upgrade (1.4 → 1.7 → 2.0)
//   - Visual editing workflows
//
// # What This Package Does NOT Claim
//
// This package does NOT claim to implement PDF 2.0, 1.7, or 1.4 fully or faithfully.
// Phase-1 covers structural open, header version, trailer /ID, InfoDict metadata,
// xref/XRef stream traversal, full-document text extraction with content operator parsing,
// multi-filter stream decoding, font encoding (WinAnsi/MacRoman/Standard/MacExpertEncoding/ToUnicode,
// /Differences array), CopyFile, and single-revision RewriteFile. CIDFont CMap,
// incremental updates, signatures, and version upgrade are not implemented.
//
// The comments in this package use "phase-1" and "not implemented" to clearly
// mark the current boundary between intent and reality.
package pdf
