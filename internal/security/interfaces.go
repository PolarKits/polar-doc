package security

import (
	"context"

	"github.com/PolarKits/polar-doc/internal/doc"
)

// SignService defines signing capability for format-specific documents.
type SignService interface {
	Sign(ctx context.Context, d doc.Document, req doc.SignRequest) (doc.SignResult, error)
}
