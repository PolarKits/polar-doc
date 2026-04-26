package ofd

import (
	"context"

	"github.com/PolarKits/polar-doc/internal/doc"
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
	doc.PageIteratorProvider
	doc.NavigatorProvider
	FirstPageInfoProvider
}

// FirstPageInfoProvider defines the interface for extracting first page information.
// OFD implementations extract PhysicalBox from Document.xml's PageArea and map it to MediaBox.
// Returns (nil, nil) when PhysicalBox is absent; returns error only on parse failure.
type FirstPageInfoProvider interface {
	FirstPageInfo(ctx context.Context, d doc.Document) (*doc.FirstPageInfoResult, error)
}

// SigningService extends Service when OFD signing is available.
type SigningService interface {
	Service
	doc.Signer
}
