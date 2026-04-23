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
//   - First-page text extraction from content streams (literal/hex strings, FlateDecode)
//
// Full semantic compatibility with ISO 32000-2 or ISO 32000-1 is NOT claimed.
// Specifically: ObjStm compressed objects are not readable; full content operator
// and font mapping are not implemented; xref/trailer/InfoDict are read but
// comprehensive integrity validation is not performed.
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
//   - Validate: checks %PDF- prefix presence (header structural check only)
//   - FirstPageInfo: traverses Catalog→Pages→Page chain, extracts MediaBox, Resources,
//     Contents, Rotate (with inheritance)
//   - xref traversal: reads traditional xref tables and XRef streams (including Prev chain);
//     Type 2 (ObjStm) entries are recognized but object stream content is not decompressed
//   - ValidateDeep: comprehensive integrity validation covering xref table/stream integrity,
//     object accessibility, trailer validity, and cross-reference consistency
//   - ExtractText: extracts literal/hex strings from first-page content streams
//     (FlateDecode supported); content operators and font mapping not implemented
//   - CopyFile: raw byte copy to destination path
//   - RewriteFile: normalizes incremental PDFs to single-revision output; follows full
//     xref chain, writes sequential live objects with fresh byte offsets, emits compact
//     xref table and fresh trailer; ObjStm objects are preserved but not expanded
//
// Not implemented in phase-1 (future work):
//   - ObjStm internal object decompression and parsing
//   - Full content operator parsing and font mapping
//   - Full-document text extraction (beyond first page)
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
// xref/XRef stream traversal, first-page content stream text extraction, CopyFile,
// and single-revision RewriteFile. Full content model, font handling, incremental
// updates, signatures, and version upgrade are not implemented.
//
// The comments in this package use "phase-1" and "not implemented" to clearly
// mark the current boundary between intent and reality.
package pdf
