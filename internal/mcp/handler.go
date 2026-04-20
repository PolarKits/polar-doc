package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/PolarKits/polardoc/internal/app"
	"github.com/PolarKits/polardoc/internal/doc"
)

const ToolNameFirstPageInfo = "pdf_first_page_info"
const ToolNameDocumentInfo = "document_info"

type FirstPageInfoInput struct {
	Path string `json:"path"`
}

type FirstPageInfoOutput struct {
	Path      string        `json:"path"`
	PagesRef  doc.RefInfo   `json:"pages_ref"`
	PageRef   doc.RefInfo   `json:"page_ref"`
	Parent    doc.RefInfo   `json:"parent"`
	MediaBox  []float64     `json:"media_box"`
	Resources doc.RefInfo   `json:"resources"`
	Contents  []doc.RefInfo `json:"contents"`
	Rotate    *int64        `json:"rotate,omitempty"`
}

type FirstPageHandler struct {
	resolver app.ServiceResolver
}

func NewFirstPageHandler(resolver app.ServiceResolver) *FirstPageHandler {
	return &FirstPageHandler{resolver: resolver}
}

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

type DocumentInfoInput struct {
	Path string `json:"path"`
}

type DocumentInfoOutput struct {
	Format          doc.Format  `json:"format"`
	Path            string      `json:"path"`
	SizeBytes       int64       `json:"size_bytes"`
	DeclaredVersion string      `json:"declared_version,omitempty"`
	PageCount       int         `json:"page_count,omitempty"`
	FileIdentifiers []string    `json:"file_identifiers,omitempty"`
	Title           string     `json:"title,omitempty"`
	Author          string     `json:"author,omitempty"`
	Creator         string     `json:"creator,omitempty"`
	Producer        string     `json:"producer,omitempty"`
}

type DocumentInfoHandler struct {
	resolver app.ServiceResolver
}

func NewDocumentInfoHandler(resolver app.ServiceResolver) *DocumentInfoHandler {
	return &DocumentInfoHandler{resolver: resolver}
}

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

func detectFormatByExtension(path string) (doc.Format, error) {
	ext := path[len(path)-4:]
	switch ext {
	case ".pdf":
		return doc.FormatPDF, nil
	case ".ofd":
		return doc.FormatOFD, nil
	default:
		return "", fmt.Errorf("unsupported file extension: %s", ext)
	}
}
