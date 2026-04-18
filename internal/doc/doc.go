// Package doc defines shared document capabilities and cross-format contracts.
//
// This package is intentionally capability-oriented. It should not flatten PDF and OFD
// semantics into one unified model.
//
// # Standards Baseline
//
// This package does NOT implement any format standard. It defines capability contracts
// (interfaces and transport types) that format-specific implementations must satisfy.
// The contracts here are named after operations, not by reference to ISO or GB/T clauses.
//
// PDF standard context: ISO 32000-2:2020 (PDF 2.0) is the baseline. Implementations
// in internal/pdf must map their semantics to the appropriate ISO 32000-2 clauses;
// this package only defines the capability surface, not the standard coverage.
//
// OFD standard context: GB/T 33190-2016 is the baseline. Implementations in internal/ofd
// must map their semantics to the appropriate GB/T 33190-2016 clauses;
// this package only defines the capability surface, not the standard coverage.
//
// # Phase-1 Scope
//
// internal/doc is a pure contract layer during phase-1 bootstrap. It contains no
// concrete parsing, validation, or rendering logic. Format-specific standards compliance
// is the responsibility of internal/pdf and internal/ofd.
package doc
