# PDF Domain

## Purpose

The PDF domain provides PDF-specific parsing, validation, repair, and writing for PolarDoc.
It keeps PDF behavior explicit and isolated from OFD.

## Responsibilities

### Current Implementation (Phase-1)

- open PDF files and read header declared version
- extract Info metadata (Title, Author, Creator, Producer) from Info dictionary
- read trailer /ID FileIdentifiers
- read PageCount from /Pages /Count
- FirstPageInfo: traverse Pages tree to first page, extract MediaBox, Resources, Contents, Rotate
- traverse xref tables and XRef streams (traditional xref + cross-reference streams)
- first-page text extraction: extract literal/hex strings from content streams (FlateDecode supported)
- CopyFile: raw byte copy to destination path
- RewriteFile: normalize incremental PDFs to single-revision output (preserves ObjStm without expansion)
- validate: check header presence (structural minimum)

### Future Responsibilities (Not Yet Implemented)

- full PDF content operator parsing and font mapping
- complete document text extraction (beyond first page)
- preview rendering and thumbnail generation
- digital signature parsing and validation
- full writer pipeline with incremental update support
- PDF version upgrade (1.4 → 1.7 → 2.0)
- advanced repair for corrupted structures

## Component Separation

### Implemented

- **parser**: reads header, xref/trailer (traditional and XRef streams), Info dictionary, first page structure
- **model**: stores PDF-native structures (FirstPageInfo, PDFRef, PDFDict) for phase-1 operations
- **text extractor**: extracts literal/hex strings from first-page content streams (FlateDecode support)
- **validator**: checks header presence (structural minimum)
- **writer (minimal)**: CopyFile (raw byte copy), RewriteFile (normalize to single-revision)

### Not Yet Implemented

- **full parser**: complete content operator parsing, font mapping, graphics state
- **full writer**: incremental updates, content modification, metadata editing, version upgrade
- **preview renderer**: thumbnail generation and rendering pipeline
- **signing**: digital signature parsing and validation
- **repair**: bounded recovery for corrupted structures beyond current handling

## Version Compatibility

Use two version views:

- declared version: value declared in the PDF header
- effective version: minimum version implied by detected features

Read policy: permissive.

- accept parseable files even when declared and effective versions differ
- expose both versions to callers

Write policy: conservative.

- emit the lowest safe version for the produced features
- avoid unnecessary version bumps

## Minimal Phase-1 Milestone (DELIVERED)

- ✅ open a PDF file and read header declared version
- ✅ inspect core structure (header, trailer, xref/XRef streams, Info dict)
- ✅ extract Info metadata (Title, Author, Creator, Producer)
- ✅ read trailer /ID FileIdentifiers and PageCount
- ✅ FirstPageInfo: traverse Pages tree, extract MediaBox, Resources, Contents, Rotate
- ✅ extract basic text from first-page content streams (literal/hex strings, FlateDecode)
- ✅ CopyFile: raw byte copy to destination
- ✅ RewriteFile: normalize incremental PDFs to single-revision output

**Current Boundaries:**
- Object streams (ObjStm): Type 2 entries recognized in XRef, but internal compressed objects not readable
- Text extraction: first-page only; content operators, font mapping, and layout not implemented
- Writer: RewriteFile produces single-revision output only; incremental updates not supported

## Phase-1 Non-Goals (Current Limitations)

- full PDF specification coverage (complete content model, operators, graphics)
- object stream (ObjStm) internal object decompression and parsing
- full-document text extraction (beyond first page)
- content operator parsing and font mapping
- preview rendering and thumbnail generation
- digital signature parsing and validation
- incremental update writer pipeline
- PDF version upgrade (1.4 → 1.7 → 2.0)
- visual editing workflows
