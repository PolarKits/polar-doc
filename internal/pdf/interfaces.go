package pdf

import (
	"context"

	"github.com/PolarKits/polardoc/internal/doc"
)

// Service defines PDF capability implementations.
//
// This package implements doc contracts with PDF semantics.
// It must not depend on the OFD package.
type Service interface {
	doc.Opener
	doc.InfoProvider
	doc.Validator
	doc.TextExtractor
	doc.PreviewRenderer
	FirstPageInfoProvider
}

// FirstPageInfoProvider returns structured first page information for a PDF document.
type FirstPageInfoProvider interface {
	FirstPageInfo(ctx context.Context, d doc.Document) (*doc.FirstPageInfoResult, error)
}

// SigningService extends Service when PDF signing is available.
type SigningService interface {
	Service
	doc.Signer
}
