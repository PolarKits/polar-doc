package doc

// Format identifies a document format domain.
type Format string

const (
	// FormatPDF identifies PDF documents.
	FormatPDF Format = "pdf"
	// FormatOFD identifies OFD documents.
	FormatOFD Format = "ofd"
)

// DocumentRef is a lightweight handle that points to a document input.
//
// It intentionally carries only routing and identity metadata.
// It is not a shared in-memory document model.
type DocumentRef struct {
	Format Format
	Path   string
}

// InfoResult is minimal document metadata returned by the info command.
type InfoResult struct {
	Format          Format
	Path            string
	SizeBytes       int64
	DeclaredVersion string
}

// ValidationReport is a structured validation output.
type ValidationReport struct {
	Valid  bool
	Errors []string
}

// PreviewRequest describes a requested preview rendering.
type PreviewRequest struct {
	Page int
	DPI  int
}

// PreviewResult describes produced preview metadata.
type PreviewResult struct {
	MediaType string
	Data      []byte
}

// TextResult describes extracted text output.
type TextResult struct {
	Text string
}

// SignRequest describes a signing request at the capability layer.
type SignRequest struct {
	Profile string
	Reason  string
}

// SignResult describes signature metadata returned after signing.
type SignResult struct {
	Method    string
	Signature []byte
}
