// Package render contains rendering orchestration and output pipelines.
//
// Rendering is a cross-cutting concern that depends on format-specific document
// content. This package defines the rendering engine contract (Engine interface)
// and coordinates output pipelines.
//
// # Phase-1 Scope
//
// Phase-1: rendering is not implemented. The Engine interface is defined but
// internal/pdf and internal/ofd both return errors indicating preview is unsupported.
// No rendering pipeline, page selection logic, or output format conversion exists yet.
//
// Future work: rendering will require format-specific content parsing (page tree,
// content streams for PDF; page structure and content objects for OFD), none of
// which are covered by current phase-1 format packages.
package render
