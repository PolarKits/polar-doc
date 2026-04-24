package pdf

import (
	"context"

	"github.com/PolarKits/polar-doc/internal/doc"
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
	doc.PageIteratorProvider
	doc.NavigatorProvider
	FirstPageInfoProvider
	FeaturesProvider
	WarningsProvider
}

// FirstPageInfoProvider returns structured first page information for a PDF document.
type FirstPageInfoProvider interface {
	FirstPageInfo(ctx context.Context, d doc.Document) (*doc.FirstPageInfoResult, error)
}

// FeaturesProvider exposes the structural feature flags detected when a PDF
// document was opened. These flags describe the xref format, encryption state,
// linearization, and version information without requiring a full xref load.
type FeaturesProvider interface {
	DocumentFeatures(ctx context.Context, d doc.Document) (PDFFeatureSet, error)
}

// WarningsProvider exposes the compat fix events recorded when a PDF document
// was opened or parsed. Callers may surface these to users or logging systems.
type WarningsProvider interface {
	Warnings(ctx context.Context, d doc.Document) ([]CompatWarning, error)
}

// Saver defines the capability to save a PDF document to a destination path.
type Saver interface {
	Save(ctx context.Context, ref doc.DocumentRef, dst string) error
}

// SigningService extends Service when PDF signing is available.
type SigningService interface {
	Service
	doc.Signer
}
