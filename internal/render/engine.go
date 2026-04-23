package render

import (
	"context"
	"fmt"

	"github.com/PolarKits/polar-doc/internal/doc"
	"github.com/PolarKits/polar-doc/internal/ofd"
	"github.com/PolarKits/polar-doc/internal/pdf"
)

type pdfEngine struct {
	svc pdf.Service
}

type ofdEngine struct {
	svc ofd.Service
}

// NewPDFEngine returns an Engine backed by a PDF service.
func NewPDFEngine(svc pdf.Service) Engine {
	return &pdfEngine{svc: svc}
}

// NewOFDEngine returns an Engine backed by an OFD service.
func NewOFDEngine(svc ofd.Service) Engine {
	return &ofdEngine{svc: svc}
}

func (e *pdfEngine) RenderPreview(ctx context.Context, d doc.Document, req doc.PreviewRequest) (doc.PreviewResult, error) {
	return e.svc.RenderPreview(ctx, d, req)
}

func (e *ofdEngine) RenderPreview(ctx context.Context, d doc.Document, req doc.PreviewRequest) (doc.PreviewResult, error) {
	return e.svc.RenderPreview(ctx, d, req)
}

// RegisterPDFEngine registers a PDF service as the engine for doc.FormatPDF.
func RegisterPDFEngine(services map[doc.Format]Engine, svc pdf.Service) {
	services[doc.FormatPDF] = NewPDFEngine(svc)
}

// RegisterOFDEngine registers an OFD service as the engine for doc.FormatOFD.
func RegisterOFDEngine(services map[doc.Format]Engine, svc ofd.Service) {
	services[doc.FormatOFD] = NewOFDEngine(svc)
}

// FormatEngines manages a mapping from document format to rendering engine.
type FormatEngines struct {
	engines map[doc.Format]Engine
}

// NewFormatEngines returns an empty registry for format engines.
func NewFormatEngines() *FormatEngines {
	return &FormatEngines{
		engines: make(map[doc.Format]Engine),
	}
}

// Register adds an engine for the given format. It returns an error if an
// engine is already registered for that format.
func (fe *FormatEngines) Register(format doc.Format, engine Engine) error {
	if _, ok := fe.engines[format]; ok {
		return fmt.Errorf("engine for format %q already registered", format)
	}
	fe.engines[format] = engine
	return nil
}

// Engine returns the registered engine for the given format and true, or
// false if no engine is registered for that format.
func (fe *FormatEngines) Engine(format doc.Format) (Engine, bool) {
	engine, ok := fe.engines[format]
	return engine, ok
}

// RenderPreview dispatches rendering to the engine matching the document format.
// It returns an error if no engine is registered for that format.
func (fe *FormatEngines) RenderPreview(ctx context.Context, d doc.Document, req doc.PreviewRequest) (doc.PreviewResult, error) {
	engine, ok := fe.Engine(d.Ref().Format)
	if !ok {
		return doc.PreviewResult{}, fmt.Errorf("no engine registered for format %q", d.Ref().Format)
	}
	return engine.RenderPreview(ctx, d, req)
}