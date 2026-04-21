# Standards Coverage Audit

## Preamble

This document audits PolarDoc's current implementation against its stated format standards and identifies gaps between the baseline standards and phase-1 code reality.

**Current code reality: phase-1 bootstrap. Internal packages contain foundational read capabilities with partial text extraction for both PDF and OFD. Do NOT read this document as claiming full standard compliance — it explicitly lists what is NOT covered.**

---

## Standards Baseline

| Format | Standard | Version |
|--------|----------|---------|
| PDF | ISO 32000-2:2020 | PDF 2.0 (primary) |
| PDF | ISO 32000-1:2008 | PDF 1.7 (compatible read) |
| PDF | ISO 32000-1:2005 | PDF 1.4 (compatible read) |
| OFD | GB/T 33190-2016 | 版式文档格式 (primary) |

### Compatibility Goals (Read)

- **PDF**: Read-only compatibility for PDF 2.0, 1.7, 1.4 files. "Compatible read" at phase-1 means: the file can be opened, its header declared version read, xref/XRef streams traversed (XRef stream format decoded, including recognition of ObjStm type entries), trailer /ID read, and Catalog→Pages→first Page chain traversed to extract Page metadata. Object stream (ObjStm) internal content is not parsed or decompressed. See internal/pdf Coverage Assessment for full details.
- **OFD**: Read-only compatibility for OFD files conforming to GB/T 33190-2016.

### Compatibility Goals (Write / Upgrade)

- **PDF version upgrade**: Converting older PDF → newer PDF (e.g. PDF 1.4 → PDF 2.0) on output is a **future capability, not implemented**.
- **OFD version upgrade**: OFD version upgrade on output is a **future capability, not implemented**.

---

## internal/app

### Role

Application wiring layer. Depends on doc contracts, resolves format services, contains no format logic or standard semantics.

### Coverage Assessment

- **Status**: NOT a standards-implementing package. No standard clauses apply.
- **Current implementation**: ServiceResolver dispatch (ByFormat) and ServiceSet composition.

### Gaps

None at this layer — it is not a standards implementation layer.

---

## internal/doc

### Role

Shared capability contracts (interfaces) and transport types. Defines what format implementations must satisfy, but does not implement any standard directly.

### Coverage Assessment

- **Status**: Contract-only. No standard semantics are implemented here.
- **current implementation**: 5 capability interfaces (Opener, InfoProvider, Validator, TextExtractor, PreviewRenderer) + optional Signer. Transport structs carry minimal metadata.
- **Version strategy**: Not reflected in this layer. The doc layer does not model version semantics; version handling is delegated to format implementations.

### Gaps

- `TextExtractor`:
  - **PDF**: partial implementation — extracts literal strings and hex strings from first-page content streams with FlateDecode decompression support. Not a complete text model (content operators not fully parsed, extraction limited to first page).
  - **OFD**: implemented — traverses Document.xml page list and extracts TextCode elements from each page's Content.xml per GB/T 33190-2016 page block semantics.
- `PreviewRenderer`: stub — returns error for both PDF and OFD. No rendering pipeline.
- `Signer`: stub — not implemented for either format.

### Notable

The doc layer explicitly does NOT define a unified document model. PDF and OFD semantics remain separate at the contract level.

---

## internal/pdf

### Role

PDF format-specific domain implementation. Maps to ISO 32000-2:2020 (PDF 2.0).

### Standards Baseline

ISO 32000-2:2020 §7 (Document Structure), §8 (File Structure), §12.8 (Digital Signatures), §14.8 (Text Extraction).

### Current Phase-1 Implementation

#### Covered

| Capability | What It Does | ISO 32000-2 Mapping |
|------------|--------------|---------------------|
| Open | Opens file handle | §7.1 (File Header / Header) |
| Info | Reads header version, trailer /ID, Info dict metadata, page count | §7.1, §7.7.2, §8.6, §8.7 |
| Validate | Checks %PDF- prefix presence | §7.1 rule only |
| ReadFirstPageInfo | Traverses Catalog→Pages→Page, extracts Page metadata | §7.7, §7.7.2, §7.8, §8.2 (partial) |
| MediaBox inheritance | Reads /MediaBox from Page or ancestor Pages | §7.7 (inheritable attribute) |
| Resources inheritance | Reads /Resources from Page or ancestor Pages | §7.7 (inheritable attribute) |
| Rotate inheritance | Reads /Rotate from Page or ancestor Pages | §7.7 (inheritable attribute) |
| ExtractText | Extracts literal/hex strings from first-page content streams (FlateDecode supported) | §8.5 (partial), §8.8 (partial), §14.8 (partial) |

#### NOT Covered (Phase-1 + Future)

| ISO 32000-2 Clause | Feature | Status |
|--------------------|---------|--------|
| §7.7 | Cross-reference table (xref) | Traditional xref + XRef stream decoding implemented; ObjStm entries recognized but not parsed |
| §7.7.2 | Trailer and trailer dictionary | Basic parsing + /ID byte strings extracted |
| §7.7.3 | Object streams | Not implemented — ObjStm type entries recognized in XRef, but object stream content is neither parsed nor decompressed |
| §7.7.4 | Incremental updates | Not implemented |
| §7.7.5 | Linearized PDF | Not implemented |
| §7.8 | File trailer / startxref | startxref keyword parsing; XRef stream decoding implemented |
| §8.2 | Object structure | Indirect object reading + XRef stream traversal; ObjStm compressed objects not readable |
| §8.3 | Strings, numbers, booleans, arrays | Primitives parsed; byte strings in arrays not handled |
| §8.4 | Names and dictionaries | Basic parsing only |
| §8.5 | Streams and filters | Partial — FlateDecode decompression for content streams; other filters not implemented |
| §8.6 | Document information dictionary | Implemented — reads Title, Author, Creator, Producer from Info dictionary |
| §8.7 | File identifiers | Implemented — reads /ID array from trailer (both traditional xref and XRef streams) |
| §8.8 | Content streams and operators | Partial — content streams located and decompressed (FlateDecode); literal/hex string extraction implemented; full operator parsing and text layout not implemented |
| §8.9 | Metadata streams | Not implemented |
| §9.1–§9.6 | Color spaces | Not implemented |
| §10.1–§10.10 | Graphics | Not implemented |
| §11.1–§11.7 | Text | Not implemented |
| §12.1–§12.7 | Interactive features | Not implemented |
| **§12.8** | **Digital signatures** | **Not implemented** |
| §13.1–§13.12 | Multimedia | Not implemented |
| §14.1–§14.12 | Marked text / accessibility | Not implemented |
| **§14.8** | **Text extraction** | **Partial** — literal/hex strings from content streams; not complete content operator parsing |

### Document Type Validation

Current: `ExtractText` and `RenderPreview` validate that the passed `Document` is the concrete `*pdf.document` type. They return "unsupported document type %T" error when cross-format document is passed (e.g. OFD doc passed to PDF service).

This check was added to prevent silent failure on cross-format misuse.

### Gaps

1. **Object streams (ObjStm)**: Type 2 entries are recognized in XRef streams, but object stream content is neither decompressed nor parsed. Objects stored in ObjStm cannot be read.
2. **Pages tree complexity**: Some PDFs have Pages trees that cannot be fully traversed with current parser.
3. **Content stream limitations**: Only FlateDecode filter supported; full content operator parsing, font mapping, and text layout analysis not implemented.
4. **No preview rendering**: PreviewRenderer returns error.
5. **No signing**: Signer capability is a stub.
6. **No writer/upgrade pipeline**: Converting PDF 1.4 → PDF 2.0 is not implemented.
7. **Known-bad samples**: Some PDFs have corrupted XRef but valid structure. These are handled via explicit test assertions (see TestPDFKnownBad_Version8x).

### PDF Version Policy

This section defines the explicit policy for PDF version handling.

#### Declared Version

The `DeclaredVersion` field in `InfoResult` is read from the PDF header comment (`%PDF-X.Y`). It reflects what the file self-declares, not a parse-validated version against ISO 32000-2 or ISO 32000-1.

The header version comment is distinct from the actual PDF version semantics defined in ISO 32000-2 §7.1. A `%PDF-1.4` header means the file claims to be PDF 1.4, but the current code does not validate whether the file's internal structure matches ISO 32000-1:2005.

#### Current Implementation Reality

The current code performs:
- Header-level permissive read: reads the `%PDF-X.Y` version comment from the file header and returns it as `DeclaredVersion`
- Traditional xref traversal: parses traditional xref tables and startxref to locate objects
- XRef stream decoding: parses XRef streams (compressed xref in object format), follows /Prev chain
- Trailer dictionary parsing: parses trailer dictionary to find /Root and /Size
- Catalog→Pages→Page traversal: traverses Catalog to find Pages ref, then Pages tree to first /Type /Page
- Minimum Page metadata extraction: extracts /MediaBox, /Resources, /Contents, /Rotate from Page dict (with inheritance from ancestor Pages)
- File body recovery scan: fallback scan for objects not found in XRef/XRef streams

Limitations:
- Object streams (ObjStm): Type 2 entries are recognized in XRef streams, but object stream content is neither decompressed nor parsed
- Some Pages tree structures cannot be fully traversed
- Content stream parsing is limited to literal/hex string extraction; full operator parsing and text layout not implemented
- Known-bad samples with XRef corruption are explicitly handled via test assertions, not silently skipped

#### Read Compatibility Scope

| Version | Read Support | Scope |
|---------|-------------|-------|
| PDF 2.0 (ISO 32000-2:2020) | Header + xref/XRef stream + trailer /ID + Info dict + Page metadata + first-page text extraction | Header validated; xref/XRef streams decoded; trailer /ID extracted; Info dict read; first Page metadata extracted; content stream text extraction (limited) |
| PDF 1.7 (ISO 32000-1:2008) | Header + xref/XRef stream + trailer /ID + Info dict + Page metadata + first-page text extraction | Header validated; xref/XRef streams decoded; trailer /ID extracted; Info dict read; first Page metadata extracted; content stream text extraction (limited) |
| PDF 1.4 (ISO 32000-1:2005) | Header + xref + trailer /ID + Info dict + Page metadata + first-page text extraction | Header validated; xref traversed; trailer /ID extracted; Info dict read; first Page metadata extracted; content stream text extraction (limited) |
| Pre-1.4 | Header + xref + trailer /ID + Info dict + Page metadata + first-page text extraction | Header validated; xref traversed; trailer /ID extracted; Info dict read; first Page metadata extracted; content stream text extraction (limited) |

"Read compatibility" at this phase means: for xref-based PDFs, the file can be opened and minimum Page metadata (MediaBox, Resources, Contents, Rotate) is extracted. XRef streams are supported (format decoded, ObjStm entries recognized). Object stream (ObjStm) content is not parsed or decompressed. Full semantic compatibility with the respective ISO specification is NOT claimed. Known-bad samples with XRef corruption are explicitly handled via test assertions.

#### Write / Upgrade Strategy

Version upgrade (reading an older PDF and outputting a newer PDF version) is a **future explicit design**, not a current implementation:

- The current code does NOT perform implicit version upgrades. Reading a PDF 1.4 file and writing it back does not silently produce a PDF 2.0 file.
- Any future upgrade capability must be an **explicit opt-in** at the application or command layer, not a hidden behavior in this package.
- Examples of explicit upgrade design options include: `--output-version=2.0` flag, automatic detection of oldest-version input and upgrade on output, or separate `upgrade` subcommand.

The writer/upgrade pipeline is listed as unimplemented in the Gaps section.

### Implementation Risks

- **Header-only validation is fragile**: A file with a valid `%PDF-1.4` header but no xref/trailer passes validation, even though it is not a complete or usable PDF.
- **No incremental update support**: Even if a document is readable, modifications cannot be written back without a writer pipeline.

---

## internal/ofd

### Role

OFD format-specific domain implementation. Maps to GB/T 33190-2016.

### Standards Baseline

GB/T 33190-2016 §4 (Document structure), §5 (Document body), §6 (Page description), §7 (Resource and mapping), §8 (Font), §9 (Security / Signatures).

### Current Phase-1 Implementation

#### Covered

| Capability | What It Does | GB/T 33190-2016 Mapping |
|------------|--------------|--------------------------|
| Open | Opens ZIP package, acquires zip.Reader | §4.1 OFD package structure |
| Validate | Checks OFD.xml and Document.xml entry presence; validates DocRoot points to existing file | §4 package requirements |
| ExtractText | Traverses Document.xml page list, extracts TextCode elements from each page's Content.xml | §6 Page description, §5 Document body |

#### NOT Covered (Phase-1 + Future)

| GB/T 33190 Clause | Feature | Status |
|-------------------|---------|--------|
| §4.2 | OFD.xml structure and DocRoot | Partial — DocRoot extracted and validated |
| §4.3 | Document.xml structure | Partial — page list traversal implemented for text extraction |
| §5 | Document body / Doc_0 content model | Partial — page-level text extraction only |
| §6 | Page description language (page, template, content) | Partial — TextCode extraction only |
| §7 | Resource and mapping | Not implemented |
| §8 | Font handling | Not implemented |
| **§9** | **Digital signatures** | **Not implemented** |
| §10 | Rendering and display | Not implemented |

### Document Type Validation

Same as PDF: `ExtractText` and `RenderPreview` validate concrete `*ofd.document` type and return "unsupported document type %T" error for cross-format misuse.

### Gaps

1. **Partial XML model parsing**: OFD.xml (DocRoot extraction/validation), Document.xml (page list traversal), and Content.xml (TextCode extraction) are parsed for text extraction. Full schema validation, resource mapping, font handling, and signature verification are not implemented.
2. **No preview rendering**: PreviewRenderer returns error.
3. **No signing**: Signer capability is a stub.
4. **No writer pipeline**: Generating OFD from scratch or modifying existing OFD is not implemented.

### Implementation Risks

- Validation checks OFD.xml and Document.xml entry presence, plus validates that DocRoot points to an existing file in the package. XML schema validation and content semantics are not verified.
- Multi-page OFD is supported for text extraction (Document.xml page list traversal), but page rendering and complex page templates are not implemented.

---

## internal/render

### Role

Rendering orchestration and output pipelines (cross-cutting).

### Coverage Assessment

- **Status**: Not implemented in phase-1.
- **Engine interface**: Defined (render/interfaces.go) but not wired to any real implementation.

### Gaps

- No rendering pipeline exists.
- No page selection or pagination.
- No output format conversion (e.g. OFD → PNG, PDF → PNG).

---

## internal/security

### Role

Cryptographic signatures, trust, and policy concerns.

### Coverage Assessment

- **Status**: Not implemented in phase-1.
- **SignService interface**: Defined but not wired to any format implementation.

### Gaps

- No cryptographic operations for either PDF or OFD.
- PDF digital signatures (ISO 32000-2 §12.8) not implemented.
- OFD digital signatures (GB/T 33190-2016 §9) not implemented.
- Certificate chain validation not implemented.
- Timestamp services not implemented.

---

## Overall Conclusion

**Q: Does the current internal code fully cover ISO 32000-2:2020 or GB/T 33190-2016?**

**No. The internal code does NOT fully cover either standard. Phase-1 covers the following:**

- **PDF**: file open + header version read + header validation + xref/XRef stream traversal + trailer /ID extraction + Info dictionary (Title/Author/Creator/Producer) + first-page content stream text extraction (FlateDecode decompression, literal/hex string extraction)
- **OFD**: ZIP package open + entry presence checks + OFD.xml DocRoot extraction/validation + Document.xml page list traversal + Content.xml TextCode extraction for all pages

**Partially implemented (functional but incomplete):**
- PDF: Content streams (FlateDecode only, no other filters), text extraction (literal/hex strings only, no operator/layout model), XRef streams (format decoded, ObjStm entries recognized but not readable)
- OFD: XML content model (DocRoot, page list, TextCode only; no resource mapping, fonts, or complex layouts)

**Not implemented:**
- PDF: Full content operator parsing, font handling, graphics, color spaces, interactive features, digital signatures, writer/upgrade pipeline, preview rendering, most stream filters beyond FlateDecode
- OFD: Resource mapping, font handling, digital signatures, writer pipeline, preview rendering, full page layout engine

### Key Future Capabilities (Not Yet Implemented)

1. **Complete PDF text extraction** (ISO 32000-2 §14.8) — requires full content operator parsing, font mapping, and layout analysis
2. **PDF preview rendering** — requires content stream + image decoding
3. **PDF signing** (ISO 32000-2 §12.8) — requires cryptographic infrastructure
4. **Complete OFD text extraction** — requires full content object model parsing (TextCode is partial only)
5. **OFD preview rendering** — requires page description language support
6. **OFD signing** (GB/T 33190-2016 §9) — requires cryptographic infrastructure
7. **PDF version upgrade** (older → newer output) — requires writer pipeline
8. **OFD version upgrade** — requires writer pipeline

### Priority Recommendations

1. **High**: Complete PDF text extraction with full content operator parsing and font mapping
2. **High**: Implement PDF object stream (ObjStm) parsing and decompression to read compressed objects
3. **Medium**: Complete OFD XML content model with full resource mapping and font handling
4. **Medium**: Preview rendering pipeline (requires complete text extraction and content parsing)
5. **Low**: Signing infrastructure (depends on crypto provider decisions)
