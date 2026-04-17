package app

import "github.com/PolarKits/polardoc/internal/doc"

// FormatService defines the core capability set expected by the application layer.
type FormatService interface {
	doc.Opener
	doc.InfoProvider
	doc.Validator
	doc.TextExtractor
	doc.PreviewRenderer
}

// SigningFormatService is an optional extension when signing is supported.
type SigningFormatService interface {
	FormatService
	doc.Signer
}

// ServiceSet wires format services for application routing.
//
// The application layer depends on doc contracts, not on PDF/OFD internals.
type ServiceSet struct {
	PDF FormatService
	OFD FormatService
}

// ServiceResolver resolves a format to a service.
type ServiceResolver interface {
	ByFormat(format doc.Format) (FormatService, bool)
}
