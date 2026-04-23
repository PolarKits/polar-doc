package render

import (
	"context"

	"github.com/PolarKits/polar-doc/internal/doc"
)

// Engine renders preview outputs for format-specific documents.
type Engine interface {
	RenderPreview(ctx context.Context, d doc.Document, req doc.PreviewRequest) (doc.PreviewResult, error)
}
