package doc

import "context"

// Document is a minimal runtime handle for an opened document.
//
// Implementations are format-specific and must preserve their own semantics.
// This interface does not represent the full document object graph of any format;
// it is only a lifecycle handle for open/close and reference access.
//
// Phase-1 note: the concrete types behind this interface are intentionally opaque
// to consumers of capability contracts. Callers must not depend on internal state.
type Document interface {
	Ref() DocumentRef
	Close() error
}

// Opener opens a format-specific document handle from a reference.
//
// This maps to format-specific "open/load" operations defined in the respective
// standards. The standard clause coverage depends on the implementing package.
//
// Phase-1 coverage:
//   - PDF: partial (file handle open; PDF header version read). Does NOT cover
//     xref, trailer, object, or stream parsing per ISO 32000-2.
//   - OFD: partial (package open; OFD.xml and Document.xml presence check).
//     Does NOT cover XML object model, ID resolution, or signature per GB/T 33190-2016.
type Opener interface {
	Open(ctx context.Context, ref DocumentRef) (Document, error)
}

// InfoProvider returns minimal metadata for an opened document.
//
// This is a convenience capability that surfaces a narrow subset of document
// properties. It does not represent full document metadata as defined by
// any standard's metadata schema.
//
// Phase-1 note: Info returns declared version string read from the document header.
// For PDF this is the %PDF-X.Y version comment. For OFD there is no declared version
// in the current phase-1 implementation.
type InfoProvider interface {
	Info(ctx context.Context, d Document) (InfoResult, error)
}

// Validator validates a format-specific document handle.
//
// Validation scope is defined by the implementing package. Validation checks
// are a subset of the structural rules defined in the respective standards.
//
// Phase-1 coverage:
//   - PDF: header presence and format (ISO 32000-2 header rule).
//     Does NOT cover xref integrity, trailer dictionary, or object validity.
//   - OFD: package-level entry presence (OFD.xml, Document.xml) per GB/T 33190-2016.
//     Does NOT cover XML schema validation, ID references, or signature verification.
type Validator interface {
	Validate(ctx context.Context, d Document) (ValidationReport, error)
}

// TextExtractor extracts text from a format-specific document handle.
//
// Text extraction semantics differ significantly between PDF and OFD.
// The result shape (plain text) is uniform across formats at this contract layer,
// but the extraction rules and content ordering are format-defined.
//
// Phase-1 coverage:
//   - PDF: partial first-page-oriented extraction from supported content streams.
//     It is intentionally narrow and not a complete PDF text model.
//   - OFD: returns an explicit "not implemented" error.
//
// Future: version upgrade path (e.g. read older PDF, output newer version) is a
// planned capability that requires a writer/upgrade pipeline, not yet implemented.
type TextExtractor interface {
	ExtractText(ctx context.Context, d Document) (TextResult, error)
}

// PreviewRenderer renders a preview for a format-specific document handle.
//
// Preview output is format-defined. The contract returns a byte payload and a
// media type. Specific rendering rules (DPI, page selection, color model) are
// passed through via PreviewRequest but interpreted by the format implementation.
//
// Phase-1 coverage: returns error "preview is not implemented" for both PDF and OFD.
type PreviewRenderer interface {
	RenderPreview(ctx context.Context, d Document, req PreviewRequest) (PreviewResult, error)
}

// Signer signs a format-specific document handle.
//
// This capability is optional during early bootstrap.
//
// Phase-1 coverage: not implemented. Signing involves cryptographic operations
// and certificate chain validation as defined in ISO 32000-2 (PDF) and GB/T 33190-2016 (OFD).
type Signer interface {
	Sign(ctx context.Context, d Document, req SignRequest) (SignResult, error)
}
