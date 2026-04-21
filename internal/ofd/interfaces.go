package ofd

import (
	"context"

	"github.com/PolarKits/polardoc/internal/doc"
)

// Service defines OFD capability implementations.
//
// This package implements doc contracts with OFD semantics.
// It must not depend on the PDF package.
type Service interface {
	doc.Opener
	doc.InfoProvider
	doc.Validator
	doc.TextExtractor
	doc.PreviewRenderer
	FirstPageInfoProvider
}

// FirstPageInfoProvider defines the interface for extracting first page information.
// OFD does not support this operation; implementations always return an error.
type FirstPageInfoProvider interface {
	FirstPageInfo(ctx context.Context, d doc.Document) (*doc.FirstPageInfoResult, error)
}

// SigningService extends Service when OFD signing is available.
type SigningService interface {
	Service
	doc.Signer
}
