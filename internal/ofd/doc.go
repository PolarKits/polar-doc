// Package ofd contains OFD-specific domain logic and operations.
//
// The OFD domain must not depend on the PDF domain.
//
// # Standards Baseline
//
// This package implements OFD semantics mapped to GB/T 33190-2016
// 《电子文件存储与交换格式 版式文档》.
//
// Compatibility target: reading OFD files conforming to GB/T 33190-2016.
// The current implementation provides read-oriented partial coverage (see Phase-1
// scope below). It does not guarantee full read-only compatibility for all
// compliant OFD files.
// Write-back / version upgrade is a future capability and is NOT yet implemented.
//
// # Phase-1 Implementation Scope
//
// Implemented (phase-1):
//   - Open: acquires ZIP archive handle, reads Version from OFD.xml Version attribute,
//     parses DocRoot path, validates Document.xml entry presence
//   - Info: provides DeclaredVersion (from OFD.xml), PageCount (from Document.xml Page elements),
//     SizeBytes, Format
//   - Validate: checks OFD.xml and Document.xml entry presence; validates DocRoot
//     path resolves to an existing file in the package
//   - ExtractText: traverses Document.xml page list and extracts TextCode elements
//     from each page's Content.xml across all pages
//   - FirstPageInfo: extracts PhysicalBox from Document.xml PageArea and maps to MediaBox
//
// Not implemented in phase-1 (future work):
//   - Complete OFD XML object model (Doc_0/*.xml body elements beyond page list and TextCode)
//   - Resource mapping and resolution (Resources.xml)
//   - Signature structure parsing and validation (Signatures.xml, Seal files)
//   - Writer / package generation pipeline
//   - OFD version upgrade (older OFD → newer OFD output)
//   - Preview rendering
//   - Complex layout, font handling, and content objects beyond TextCode
//
// # What This Package Does NOT Claim
//
// This package does NOT claim to implement GB/T 33190-2016 fully or faithfully.
// The current phase-1 covers ZIP open, Version/DocRoot, page count, DocRoot integrity,
// and TextCode extraction. Full schema validation, resource mapping, signatures,
// and writing are not implemented.
//
// The comments in this package use "phase-1" and "not implemented" to clearly
// mark the current boundary between intent and reality.
package ofd
