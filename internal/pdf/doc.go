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
// "Read compatibility" means the file can be opened and its header version
// declared_version read. The current implementation performs a header-level
// permissive read only: it reads the %PDF-X.Y version comment from the file
// header. This does NOT constitute full semantic compatibility with the
// corresponding ISO 32000-2 or ISO 32000-1 specification.
//
// For PDF 1.4 and 1.7: current read support is header-level permissive read.
// The code does not validate xref tables, trailers, or object graphs; therefore
// a valid header does not guarantee the file is semantically compliant with
// ISO 32000-1:2005 or ISO 32000-1:2008. The permissive read allows opening
// files for inspection without full parse.
//
// For older versions (pre-1.4): same policy applies — header-level permissive
// read is performed, but full semantic compatibility is not claimed.
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
// Write-back or incremental update is not implemented in phase-1.
//
// # Phase-1 Implementation Scope
//
// Phase-1 coverage is intentionally minimal and structural only.
//
// Covered (phase-1):
//   - PDF file open (file handle acquisition)
//   - PDF header version read (the %PDF-X.Y comment in the file header)
//   - Header presence validation (checks the %PDF- prefix; does NOT validate
//     xref, trailer, or object integrity)
//
// NOT covered (phase-1 or future):
//   - xref table parsing and validation (ISO 32000-2 §7.7)
//   - trailer and trailer dictionary parsing (ISO 32000-2 §7.7.2)
//   - document catalog, info dictionary (ISO 32000-2 §7.7.2, §8.6)
//   - page tree, page objects (ISO 32000-2 §7.7.2, §7.7.3)
//   - content streams and operators (ISO 32000-2 §8.8)
//   - text extraction and content ordering (ISO 32000-2 §14.8)
//   - interactive features (annotations, forms, JavaScript)
//   - signature and certification (ISO 32000-2 §12.8)
//   - incremental update / writer pipeline
//
// # What This Package Does NOT Claim
//
// This package does NOT claim to implement PDF 2.0, 1.7, or 1.4 fully or faithfully.
// Specifically, the following are explicitly out of scope for phase-1 and remain
// unimplemented:
//   - Any form of PDF writing or version upgrade
//   - Full content stream parsing
//   - Cryptographic security handlers
//   - Interactive features
//
// The comments in this package use "phase-1" and "not implemented" to clearly
// mark the current boundary between intent and reality.
package pdf
