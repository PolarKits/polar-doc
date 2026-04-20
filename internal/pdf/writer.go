package pdf

import "errors"

// ErrDowngradeNotAllowed is returned when SaveOptions.TargetVersion is lower than
// the source document's effective version. Downgrading a PDF version is not supported
// because it may silently discard features required by the document's content.
var ErrDowngradeNotAllowed = errors.New("pdf: target version is lower than source version; downgrade not allowed")

// SaveStrategy controls the structural approach used when writing a PDF document.
type SaveStrategy int

const (
	// StrategyPassthrough copies source bytes to destination without parsing or
	// modification. Guarantees byte-level identity with the source. This is the
	// current Phase-1 implementation (CopyFile).
	StrategyPassthrough SaveStrategy = iota

	// StrategyIncrementalAppend appends a new revision to the end of the source
	// document without modifying its existing bytes. Preserves all prior revisions
	// and any digital signatures that cover the original byte range.
	// Suitable for metadata edits, annotation additions, and form fills.
	StrategyIncrementalAppend

	// StrategyFullRewrite rebuilds the entire document from the parsed object graph
	// and writes a clean, single-revision output at the target version.
	// Smaller output than IncrementalAppend. Discards revision history.
	// Note: invalidates any digital signatures that depend on original byte ranges.
	StrategyFullRewrite
)

// SaveOptions configures the behavior of a PDF write operation.
// Use DefaultSaveOptions or ArchiveSaveOptions as starting points rather than
// constructing SaveOptions from scratch.
type SaveOptions struct {
	// Strategy selects the structural write approach.
	Strategy SaveStrategy

	// TargetVersion is the PDF version to write. Zero value means preserve the
	// source version, with a floor of PDF17. Explicit setting is preferred.
	// Has no effect when Strategy is StrategyPassthrough.
	TargetVersion PDFVersion

	// MigrateInfoDictToXMP, when true, copies /Info dictionary fields into an
	// XMP metadata stream during StrategyFullRewrite. Recommended for PDF 2.0 output
	// because the /Info dictionary is deprecated in ISO 32000-2.
	MigrateInfoDictToXMP bool

	// UpgradeEncryption, when true, upgrades the encryption algorithm to AES-256
	// during StrategyFullRewrite if the source document is encrypted.
	// Has no effect on unencrypted documents.
	UpgradeEncryption bool

	// RemoveDeprecated, when true, removes structures that are deprecated in the
	// target version (e.g. XFA forms when targeting PDF 2.0).
	RemoveDeprecated bool
}

// DefaultSaveOptions returns the recommended save configuration for general use.
//
// Strategy: IncrementalAppend — appends a new revision without touching existing
// bytes, which preserves any digital signatures present in the source document.
// TargetVersion: PDF 1.7 — the ISO 32000-1 baseline with the broadest reader support.
func DefaultSaveOptions() SaveOptions {
	return SaveOptions{
		Strategy:      StrategyIncrementalAppend,
		TargetVersion: PDF17,
	}
}

// ArchiveSaveOptions returns the recommended save configuration for long-term archival.
//
// Strategy: FullRewrite — produces a clean, single-revision document.
// TargetVersion: PDF 2.0 — ISO 32000-2 with XMP metadata and current encryption standards.
// MigrateInfoDictToXMP: true — /Info dict is deprecated in PDF 2.0.
// UpgradeEncryption: true — ensures AES-256 if the document was encrypted.
// RemoveDeprecated: true — removes XFA and other deprecated structures.
func ArchiveSaveOptions() SaveOptions {
	return SaveOptions{
		Strategy:             StrategyFullRewrite,
		TargetVersion:        PDF20,
		MigrateInfoDictToXMP: true,
		UpgradeEncryption:    true,
		RemoveDeprecated:     true,
	}
}
