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
- full-document text extraction: extract text from content streams via content operator parsing (BT/ET blocks, Tj/TJ operators, TJ array spacing); supports WinAnsiEncoding, MacRomanEncoding, ToUnicode CMap font encoding; stream filter support: FlateDecode, ASCIIHexDecode, ASCII85Decode, LZWDecode
- CopyFile: raw byte copy to destination path
- RewriteFile: normalize incremental PDFs to single-revision output (preserves ObjStm via resolveFromObjStm)
- validate: 5-level structural validation (Header → XRef → Trailer → Catalog → Pages)
- content_parser: content stream operator parsing (text blocks, string operands, hex strings, BT/ET/Tj/TJ operators, TJ array spacing analysis)
- stream_filter: multi-filter framework (FlateDecode, ASCIIHexDecode, ASCII85Decode, LZWDecode; supports filter chains)
- font_encoding: built-in encoding tables (WinAnsiEncoding 128-255, MacRomanEncoding 128-255, StandardEncoding 128-255, /Differences array parsing, applyByteMapping for font-to-Unicode)

### Future Responsibilities (Not Yet Implemented)

- CIDFont CMap-based font encoding
- preview rendering and thumbnail generation
- digital signature parsing and validation
- incremental update writer pipeline (RewriteFile produces single-revision only)
- PDF version upgrade (1.4 → 1.7 → 2.0)
- advanced repair for corrupted structures

## Component Separation

### Implemented

- **parser**: reads header, xref/trailer (traditional and XRef streams), Info dictionary, first page structure
- **model**: stores PDF-native structures (FirstPageInfo, PDFRef, PDFDict) for phase-1 operations
- **text extractor**: full-document text extraction with content operator parsing (BT/ET, Tj/TJ, TJ spacing); supports WinAnsi/MacRoman/ToUnicode encoding; multi-filter framework
- **validator**: 5-level structural validation (Header → XRef → Trailer → Catalog → Pages)
- **writer (minimal)**: CopyFile (raw byte copy), RewriteFile (normalize to single-revision)
- **content_parser**: content stream operator parsing (text blocks, string operands, hex strings, content operators, TJ array spacing analysis)
- **stream_filter**: multi-filter framework (FlateDecode, ASCIIHexDecode, ASCII85Decode, LZWDecode; supports filter chains)
- **font_encoding**: built-in encoding tables (WinAnsiEncoding 128-255, MacRomanEncoding 128-255, StandardEncoding 128-255, /Differences array parsing, applyByteMapping for font-to-Unicode mapping)

### Not Yet Implemented

- **full writer**: incremental updates, content modification, metadata editing, version upgrade
- **preview renderer**: thumbnail generation and rendering pipeline
- **signing**: digital signature parsing and validation
- **repair**: bounded recovery for corrupted structures beyond current handling
- **CIDFont CMap**: CIDFont-based font encoding support

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
- Object streams (ObjStm): Type 2 entries resolved via resolveFromObjStm; compression is read-only
- Text extraction: full-document extraction with content operator parsing; font encoding (WinAnsi/MacRoman/StandardEncoding/ToUnicode, /Differences array); layout analysis not implemented
- Writer: RewriteFile produces single-revision output only; incremental updates not supported
- Font encoding: CIDFont CMap not supported

## Phase-1 Non-Goals (Current Limitations)

- full PDF specification coverage (complete content model, operators, graphics)
- CIDFont CMap-based font encoding
- preview rendering and thumbnail generation
- digital signature parsing and validation
- incremental update writer pipeline
- PDF version upgrade (1.4 → 1.7 → 2.0)
- visual editing workflows
