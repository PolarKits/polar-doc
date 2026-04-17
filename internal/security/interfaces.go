package security

import (
	"context"

	"github.com/PolarKits/polardoc/internal/doc"
)

// SignService defines signing capability for format-specific documents.
type SignService interface {
	Sign(ctx context.Context, d doc.Document, req doc.SignRequest) (doc.SignResult, error)
}
