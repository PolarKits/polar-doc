package app

import (
	"context"

	"github.com/PolarKits/polardoc/internal/doc"
)

// FirstPageInfoResult holds first page info result at app capability level.
type FirstPageInfoResult = doc.FirstPageInfoResult

// FormatService defines the core capability set expected by the application layer.
//
// This is a capability bundle. It does not imply full standard compliance
// for any format. Each method maps to a doc contract whose standards coverage
// is defined in the doc package and implemented by internal/pdf or internal/ofd.
type FormatService interface {
	doc.Opener
	doc.InfoProvider
	doc.Validator
	doc.TextExtractor
	doc.PreviewRenderer
	FirstPageInfoProvider
}

// FirstPageInfoProvider is the capability interface for first page info extraction.
// PDF implementation returns (*FirstPageInfoResult, nil).
// OFD implementation returns (nil, error) since OFD does not support this capability.
type FirstPageInfoProvider interface {
	FirstPageInfo(ctx context.Context, d doc.Document) (*FirstPageInfoResult, error)
}

// SigningFormatService is an optional extension when signing is supported.
//
// Signing coverage is defined in doc.Signer and implemented by format packages.
// This interface is not yet wired in phase-1.
type SigningFormatService interface {
	FormatService
	doc.Signer
}

// PDFSaver is the capability interface for saving a PDF document to a destination path.
//
// This is a PDF-specific capability. Only PDF service implements this via CopyFile.
// OFD does not support save operations. This interface exists separately
// from FormatService to maintain PDF/OFD semantic separation.
type PDFSaver interface {
	Save(ctx context.Context, ref doc.DocumentRef, dst string) error
}

// ServiceSet wires format services for application routing.
//
// The application layer depends on doc contracts, not on PDF/OFD internals.
// ServiceSet is a simple struct-based composition; it contains no routing
// logic or standard semantics.
type ServiceSet struct {
	PDF FormatService
	OFD FormatService
}

// ServiceResolver resolves a format to a service.
//
// Resolution is by format string (doc.FormatPDF, doc.FormatOFD), not by
// file content sniffing. Format is determined by file extension at the
// CLI entry point before this resolver is consulted.
type ServiceResolver interface {
	ByFormat(format doc.Format) (FormatService, bool)
}
