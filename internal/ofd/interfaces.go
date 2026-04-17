package ofd

import "github.com/PolarKits/polardoc/internal/doc"

// Service defines OFD capability implementations.
//
// This package implements doc contracts with OFD semantics.
// It must not depend on the PDF package.
type Service interface {
	doc.Opener
	doc.Validator
	doc.TextExtractor
	doc.PreviewRenderer
}

// SigningService extends Service when OFD signing is available.
type SigningService interface {
	Service
	doc.Signer
}
