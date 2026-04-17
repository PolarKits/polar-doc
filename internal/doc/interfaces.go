package doc

import "context"

// Document is a minimal runtime handle for an opened document.
//
// Implementations are format-specific and must preserve their own semantics.
type Document interface {
	Ref() DocumentRef
	Close() error
}

// Opener opens a format-specific document handle from a reference.
type Opener interface {
	Open(ctx context.Context, ref DocumentRef) (Document, error)
}

// InfoProvider returns minimal metadata for an opened document.
type InfoProvider interface {
	Info(ctx context.Context, d Document) (InfoResult, error)
}

// Validator validates a format-specific document handle.
type Validator interface {
	Validate(ctx context.Context, d Document) (ValidationReport, error)
}

// TextExtractor extracts text from a format-specific document handle.
type TextExtractor interface {
	ExtractText(ctx context.Context, d Document) (TextResult, error)
}

// PreviewRenderer renders a preview for a format-specific document handle.
type PreviewRenderer interface {
	RenderPreview(ctx context.Context, d Document, req PreviewRequest) (PreviewResult, error)
}

// Signer signs a format-specific document handle.
//
// This capability is optional during early bootstrap.
type Signer interface {
	Sign(ctx context.Context, d Document, req SignRequest) (SignResult, error)
}
