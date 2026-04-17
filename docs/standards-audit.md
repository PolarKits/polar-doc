# Standards Coverage Audit

## Preamble

This document audits PolarDoc's current implementation against its stated format standards and identifies gaps between the baseline standards and phase-1 code reality.

**Current code reality: phase-1 bootstrap. Internal packages contain minimal stubs and structural checks. Do NOT read this document as claiming full standard compliance — it explicitly lists what is NOT covered.**

---

## Standards Baseline

| Format | Standard | Version |
|--------|----------|---------|
| PDF | ISO 32000-2:2020 | PDF 2.0 (primary) |
| PDF | ISO 32000-1:2008 | PDF 1.7 (compatible read) |
| PDF | ISO 32000-1:2005 | PDF 1.4 (compatible read) |
| OFD | GB/T 33190-2016 | 版式文档格式 (primary) |

### Compatibility Goals (Read)

- **PDF**: Read-only compatibility for PDF 2.0, 1.7, 1.4 files. "Compatible read" at phase-1 means: the file can be opened, its header declared version read, and for traditional xref-based PDFs, the Catalog→Pages→first Page chain can be traversed to extract minimum Page metadata (MediaBox, Resources, Contents, Rotate). Limitations: XRef streams, trailer /ID byte strings, and object streams are not yet supported. See internal/pdf Coverage Assessment for full details.
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

- `TextExtractor`: stub — returns empty TextResult{} for both PDF and OFD. No text extraction, ordering, or content completeness guarantees.
- `PreviewRenderer`: stub — returns error for both PDF and OFD. No rendering pipeline.
- `Signer`: stub — not implemented for either format.
- `InfoResult.DeclaredVersion`: For OFD this field is always empty (no declared version in OFD header equivalent).

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
| Info | Reads header declared version (%PDF-X.Y) | §7.1 (Header) |
| Validate | Checks %PDF- prefix presence | §7.1 rule only |
| ReadFirstPageInfo | Traverses Catalog→Pages→Page, extracts Page metadata | §7.7, §7.7.2, §7.8, §8.2 (partial) |
| MediaBox inheritance | Reads /MediaBox from Page or ancestor Pages | §7.7 (inheritable attribute) |
| Resources inheritance | Reads /Resources from Page or ancestor Pages | §7.7 (inheritable attribute) |
| Rotate inheritance | Reads /Rotate from Page or ancestor Pages | §7.7 (inheritable attribute) |

#### NOT Covered (Phase-1 + Future)

| ISO 32000-2 Clause | Feature | Status |
|--------------------|---------|--------|
| §7.7 | Cross-reference table (xref) | Traditional xref + XRef stream decoding implemented; object stream decompression not implemented |
| §7.7.2 | Trailer and trailer dictionary | Basic parsing only; /ID byte strings not handled |
| §7.7.3 | Object streams | Partial: can locate stream objects, cannot decompress content |
| §7.7.4 | Incremental updates | Not implemented |
| §7.7.5 | Linearized PDF | Not implemented |
| §7.8 | File trailer / startxref | startxref keyword parsing; XRef stream decoding implemented |
| §8.2 | Object structure | Indirect object reading + XRef stream traversal; object stream content not parsed |
| §8.3 | Strings, numbers, booleans, arrays | Primitives parsed; byte strings in arrays not handled |
| §8.4 | Names and dictionaries | Basic parsing only |
| §8.5 | Streams and filters | Not implemented |
| §8.6 | Document information dictionary | Not implemented |
| §8.7 | File identifiers | Not implemented |
| §8.8 | Content streams and operators | Not implemented |
| §8.9 | Metadata streams | Not implemented |
| §9.1–§9.6 | Color spaces | Not implemented |
| §10.1–§10.10 | Graphics | Not implemented |
| §11.1–§11.7 | Text | Not implemented |
| §12.1–§12.7 | Interactive features | Not implemented |
| **§12.8** | **Digital signatures** | **Not implemented** |
| §13.1–§13.12 | Multimedia | Not implemented |
| §14.1–§14.12 | Marked text / accessibility | Not implemented |
| **§14.8** | **Text extraction** | **Not implemented** (stub returns empty) |

### Document Type Validation

Current: `ExtractText` and `RenderPreview` validate that the passed `Document` is the concrete `*pdf.document` type. They return "unsupported document type %T" error when cross-format document is passed (e.g. OFD doc passed to PDF service).

This check was added to prevent silent failure on cross-format misuse.

### Gaps

1. **Object stream content**: Cannot decompress content within object streams.
2. **Trailer /ID byte strings**: Cannot parse trailer dictionaries containing byte string arrays in the /ID field.
3. **Pages tree complexity**: Some PDFs have Pages trees that cannot be fully traversed with current parser.
4. **No content extraction**: TextExtractor returns empty stub.
5. **No preview rendering**: PreviewRenderer returns error.
6. **No signing**: Signer capability is a stub.
7. **No writer/upgrade pipeline**: Converting PDF 1.4 → PDF 2.0 is not implemented.
8. **Known-bad samples**: Some PDFs have corrupted XRef but valid structure. These are handled via explicit test assertions (see TestPDFKnownBad_Version8x).

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
- Does NOT decompress content within object streams
- Does NOT interpret trailer /ID byte string arrays (they are parsed as PDFHexString but not semantically processed)
- Some Pages tree structures cannot be fully traversed
- Known-bad samples with XRef corruption are explicitly handled via test assertions, not silently skipped

#### Read Compatibility Scope

| Version | Read Support | Scope |
|---------|-------------|-------|
| PDF 2.0 (ISO 32000-2:2020) | Header + xref/XRef stream + Page metadata | Header validated; xref traversed; XRef streams decoded; first Page metadata extracted |
| PDF 1.7 (ISO 32000-1:2008) | Header + xref/XRef stream + Page metadata | Header validated; xref traversed; XRef streams decoded; first Page metadata extracted |
| PDF 1.4 (ISO 32000-1:2005) | Header + xref + Page metadata | Header validated; xref traversed; first Page metadata extracted |
| Pre-1.4 | Header + xref + Page metadata | Header validated; xref traversed; first Page metadata extracted |

"Read compatibility" at this phase means: for xref-based PDFs, the file can be opened and minimum Page metadata (MediaBox, Resources, Contents, Rotate) is extracted. XRef streams are supported for PDFs using cross-reference streams. Object stream content decompression is not yet supported. Full semantic compatibility with the respective ISO specification is NOT claimed. Known-bad samples with XRef corruption are explicitly handled via test assertions.

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
| Validate | Checks OFD.xml and Document.xml entry presence | §4 package requirements only |

#### NOT Covered (Phase-1 + Future)

| GB/T 33190 Clause | Feature | Status |
|-------------------|---------|--------|
| §4.2 | OFD.xml structure and DocRoot | Not implemented |
| §4.3 | Document.xml structure | Not implemented |
| §5 | Document body / Doc_0 content model | Not implemented |
| §6 | Page description language (page, template, content) | Not implemented |
| §7 | Resource and mapping | Not implemented |
| §8 | Font handling | Not implemented |
| **§9** | **Digital signatures** | **Not implemented** |
| §10 | Rendering and display | Not implemented |

### Document Type Validation

Same as PDF: `ExtractText` and `RenderPreview` validate concrete `*ofd.document` type and return "unsupported document type %T" error for cross-format misuse.

### Gaps

1. **No XML model parsing**: OFD.xml and Document.xml are opened as ZIP entries but their XML content is not parsed. No page tree, no content objects, no resource mapping.
2. **No text extraction**: TextExtractor returns empty stub.
3. **No preview rendering**: PreviewRenderer returns error.
4. **No signing**: Signer capability is a stub.
5. **No writer pipeline**: Generating OFD from scratch or modifying existing OFD is not implemented.

### Implementation Risks

- Current validation only checks two filenames exist inside the ZIP. A file named `Document.xml` but with garbage content passes validation.
- No support for multi-page OFD (Doc_0/Document.xml may contain multiple Page nodes per GB/T 33190-2016 §5).

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

**No. The internal code does NOT fully cover either standard. It covers only the following in phase-1:**

- PDF: file open + header version read + header presence validation (structural minimum)
- OFD: ZIP package open + two entry name presence checks (structural minimum)

**All other standard clauses — content extraction, page rendering, signatures, writers, xref parsing, XML content model parsing — are not implemented.**

### Key Future Capabilities (Not Yet Implemented)

1. **PDF text extraction** (ISO 32000-2 §14.8) — requires content stream parsing
2. **PDF preview rendering** — requires content stream + image decoding
3. **PDF signing** (ISO 32000-2 §12.8) — requires cryptographic infrastructure
4. **OFD text extraction** — requires XML content object model parsing
5. **OFD preview rendering** — requires page description language support
6. **OFD signing** (GB/T 33190-2016 §9) — requires cryptographic infrastructure
7. **PDF version upgrade** (older → newer output) — requires writer pipeline
8. **OFD version upgrade** — requires writer pipeline

### Priority Recommendations

1. **High**: Implement PDF xref and trailer parsing to enable reliable document open for all compliant PDF files
2. **High**: Implement PDF text extraction to fulfill the ExtractText capability contract
3. **Medium**: OFD XML content model (at least Doc_0 structure) to enable proper validation beyond entry name checks
4. **Medium**: Preview rendering pipeline (requires text extraction first)
5. **Low**: Signing infrastructure (depends on crypto provider decisions)
