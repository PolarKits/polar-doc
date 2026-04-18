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
// Read-only compatibility is guaranteed for compliant OFD files.
// Write-back / version upgrade is a future capability and is NOT yet implemented.
//
// # Phase-1 Implementation Scope
//
// Phase-1 coverage is intentionally minimal and structural only.
//
// Covered (phase-1):
//   - OFD package open (ZIP archive handle acquisition)
//   - OFD.xml entry presence check (root document container)
//   - Document.xml entry presence check (primary document entry)
//
// NOT covered (phase-1 or future):
//   - OFD XML object model parsing (Doc_0/*.xml body elements)
//   - Page structure, template pages, public/residents
//   - Text, image, path, pen, brush content objects
//   - ID and reference resolution across OFD XML entries
//   - Signatures and security (GB/T 33190-2016 §9)
//   - Digital signature verification
//   - Writer / package generation pipeline
//   - OFD version upgrade (older OFD → newer OFD output)
//
// # What This Package Does NOT Claim
//
// This package does NOT claim to implement GB/T 33190-2016 fully or faithfully.
// The current phase-1 only opens the ZIP package and checks two entry names.
// All content model, page description, and security features are not implemented.
//
// The comments in this package use "phase-1" and "not implemented" to clearly
// mark the current boundary between intent and reality.
package ofd
