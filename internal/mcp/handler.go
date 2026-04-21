package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PolarKits/polardoc/internal/app"
	"github.com/PolarKits/polardoc/internal/doc"
)

// ToolNameFirstPageInfo is the MCP tool name for retrieving first-page structure info.
const ToolNameFirstPageInfo = "pdf_first_page_info"

// ToolNameDocumentInfo is the MCP tool name for retrieving document-level metadata.
const ToolNameDocumentInfo = "document_info"

// FirstPageInfoInput is the payload for the pdf_first_page_info tool.
type FirstPageInfoInput struct {
	// Path is the file system path to the PDF document.
	Path string `json:"path"`
}

// FirstPageInfoOutput is the result for the pdf_first_page_info tool.
type FirstPageInfoOutput struct {
	// Path is the file system path to the document.
	Path string `json:"path"`
	// PagesRef is the indirect reference to the root Pages object.
	PagesRef doc.RefInfo `json:"pages_ref"`
	// PageRef is the indirect reference to the first page object.
	PageRef doc.RefInfo `json:"page_ref"`
	// Parent is the indirect reference to the parent Pages object.
	Parent doc.RefInfo `json:"parent"`
	// MediaBox is the page media box rectangle [llx, lly, urx, ury].
	MediaBox []float64 `json:"media_box"`
	// Resources is the indirect reference to the resource dictionary.
	Resources doc.RefInfo `json:"resources"`
	// Contents is a slice of indirect references to content streams.
	Contents []doc.RefInfo `json:"contents"`
	// Rotate is the page rotation in degrees (0, 90, 180, 270). Nil if not specified.
	Rotate *int64 `json:"rotate,omitempty"`
}

// FirstPageHandler handles the pdf_first_page_info MCP tool.
type FirstPageHandler struct {
	resolver app.ServiceResolver
}

// NewFirstPageHandler creates a handler for the pdf_first_page_info tool.
func NewFirstPageHandler(resolver app.ServiceResolver) *FirstPageHandler {
	return &FirstPageHandler{resolver: resolver}
}

// Handle implements the ToolHandler interface for the pdf_first_page_info tool.
// It extracts first page structure information from a PDF document.
func (h *FirstPageHandler) Handle(ctx context.Context, tool string, payload []byte) ([]byte, error) {
	if tool != ToolNameFirstPageInfo {
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}

	var input FirstPageInfoInput
	if err := json.Unmarshal(payload, &input); err != nil {
		return nil, fmt.Errorf("invalid input JSON: %w", err)
	}

	if input.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	format, err := detectFormatByExtension(input.Path)
	if err != nil {
		return nil, err
	}

	svc, ok := h.resolver.ByFormat(format)
	if !ok {
		return nil, fmt.Errorf("no service for format %q", format)
	}

	d, err := svc.Open(ctx, doc.DocumentRef{Format: format, Path: input.Path})
	if err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}
	defer d.Close()

	result, err := svc.FirstPageInfo(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("first page info: %w", err)
	}

	output := FirstPageInfoOutput{
		Path:      input.Path,
		PagesRef:  result.PagesRef,
		PageRef:   result.PageRef,
		Parent:    result.Parent,
		MediaBox:  result.MediaBox,
		Resources: result.Resources,
		Contents:  result.Contents,
		Rotate:    result.Rotate,
	}

	return json.Marshal(output)
}

// ToolNameDocumentValidate is the MCP tool name for document validation.
const ToolNameDocumentValidate = "document_validate"

// DocumentInfoInput is the payload for the document_info tool.
type DocumentInfoInput struct {
	// Path is the file system path to the document.
	Path string `json:"path"`
}

// DocumentInfoOutput is the result for the document_info tool.
type DocumentInfoOutput struct {
	// Format is the document format domain (PDF or OFD).
	Format doc.Format `json:"format"`
	// Path is the file system path to the document.
	Path string `json:"path"`
	// SizeBytes is the file size in bytes.
	SizeBytes int64 `json:"size_bytes"`
	// DeclaredVersion is the format version declared in the document header.
	DeclaredVersion string `json:"declared_version,omitempty"`
	// PageCount is the number of pages in the document.
	PageCount int `json:"page_count,omitempty"`
	// FileIdentifiers is the list of file identifiers (PDF only; OFD returns empty).
	FileIdentifiers []string `json:"file_identifiers,omitempty"`
	// Title is the document title from metadata (PDF only; OFD returns empty).
	Title string `json:"title,omitempty"`
	// Author is the document author from metadata (PDF only; OFD returns empty).
	Author string `json:"author,omitempty"`
	// Creator is the document creator from metadata (PDF only; OFD returns empty).
	Creator string `json:"creator,omitempty"`
	// Producer is the document producer from metadata (PDF only; OFD returns empty).
	Producer string `json:"producer,omitempty"`
}

// DocumentInfoHandler handles the document_info MCP tool.
type DocumentInfoHandler struct {
	resolver app.ServiceResolver
}

// NewDocumentInfoHandler creates a handler for the document_info tool.
func NewDocumentInfoHandler(resolver app.ServiceResolver) *DocumentInfoHandler {
	return &DocumentInfoHandler{resolver: resolver}
}

// Handle implements the ToolHandler interface for the document_info tool.
// It extracts document-level metadata from PDF or OFD documents.
func (h *DocumentInfoHandler) Handle(ctx context.Context, tool string, payload []byte) ([]byte, error) {
	if tool != ToolNameDocumentInfo {
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}

	var input DocumentInfoInput
	if err := json.Unmarshal(payload, &input); err != nil {
		return nil, fmt.Errorf("invalid input JSON: %w", err)
	}

	if input.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	format, err := detectFormatByExtension(input.Path)
	if err != nil {
		return nil, err
	}

	svc, ok := h.resolver.ByFormat(format)
	if !ok {
		return nil, fmt.Errorf("no service for format %q", format)
	}

	d, err := svc.Open(ctx, doc.DocumentRef{Format: format, Path: input.Path})
	if err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}
	defer d.Close()

	info, err := svc.Info(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("document info: %w", err)
	}

	output := DocumentInfoOutput{
		Format:          info.Format,
		Path:            info.Path,
		SizeBytes:       info.SizeBytes,
		DeclaredVersion: info.DeclaredVersion,
		PageCount:       info.PageCount,
		FileIdentifiers: info.FileIdentifiers,
		Title:           info.Title,
		Author:          info.Author,
		Creator:         info.Creator,
		Producer:        info.Producer,
	}

	return json.Marshal(output)
}

// DocumentValidateInput is the payload for the document_validate tool.
type DocumentValidateInput struct {
	// Path is the file system path to the document.
	Path string `json:"path"`
}

// DocumentValidateOutput is the result for the document_validate tool.
type DocumentValidateOutput struct {
	// Valid is true when the document passes basic structural checks for its format.
	Valid bool `json:"valid"`
	// Errors contains human-readable structural failure reasons.
	// This is not an exhaustive list of standard violations.
	Errors []string `json:"errors,omitempty"`
}

// DocumentValidateHandler handles the document_validate MCP tool.
type DocumentValidateHandler struct {
	resolver app.ServiceResolver
}

// NewDocumentValidateHandler creates a handler for the document_validate tool.
func NewDocumentValidateHandler(resolver app.ServiceResolver) *DocumentValidateHandler {
	return &DocumentValidateHandler{resolver: resolver}
}

// Handle implements the ToolHandler interface for the document_validate tool.
// It validates the structural integrity of a PDF or OFD document.
func (h *DocumentValidateHandler) Handle(ctx context.Context, tool string, payload []byte) ([]byte, error) {
	if tool != ToolNameDocumentValidate {
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}

	var input DocumentValidateInput
	if err := json.Unmarshal(payload, &input); err != nil {
		return nil, fmt.Errorf("invalid input JSON: %w", err)
	}

	if input.Path == "" {
		return nil, fmt.Errorf("path is required")
	}

	format, err := detectFormatByExtension(input.Path)
	if err != nil {
		return nil, err
	}

	svc, ok := h.resolver.ByFormat(format)
	if !ok {
		return nil, fmt.Errorf("no service for format %q", format)
	}

	d, err := svc.Open(ctx, doc.DocumentRef{Format: format, Path: input.Path})
	if err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}
	defer d.Close()

	report, err := svc.Validate(ctx, d)
	if err != nil {
		return nil, fmt.Errorf("validate document: %w", err)
	}

	return json.Marshal(report)
}

func detectFormatByExtension(path string) (doc.Format, error) {
	return doc.DetectFormatByExtension(path)
}
