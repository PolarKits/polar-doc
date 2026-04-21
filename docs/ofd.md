# OFD Domain

## Purpose

The OFD domain provides OFD-specific container, XML, signature, validation, and writing behavior for PolarDoc.
It keeps OFD semantics explicit and separate from PDF internals.

## Responsibilities

### Current Implementation (Phase-1)

- open and manage OFD ZIP package contents
- read OFD.xml Version attribute
- parse DocRoot and validate Document.xml exists
- count pages from Document.xml Page elements
- extract text from TextCode elements across all pages via Content.xml
- validate package structure (OFD.xml and Document.xml presence, DocRoot reference integrity)

### Future Responsibilities (Not Yet Implemented)

- full OFD XML model parsing (complete schema structures beyond basic page enumeration)
- resource mapping and resolution (Resources.xml, multi-document references)
- signature structure parsing and validation (Signatures.xml, Seal files)
- write valid OFD package output (serialization and modification)
- preview rendering and first page inspection
- complex layout and font handling

## Core Components

### Implemented

- **container**: reads package entries and normalized paths; provides access to OFD ZIP contents
- **basic XML reader**: parses OFD.xml (Version, DocRoot) and Document.xml (Page list, page count)
- **text extractor**: extracts TextCode elements from page Content.xml files across all pages
- **validator**: checks package structure (OFD.xml, Document.xml presence) and DocRoot reference integrity

### Not Yet Implemented

- **full XML model**: complete OFD schema structures (beyond basic page enumeration)
- **resolver**: reference resolution across document, page, and resource scopes
- **signature**: parsing and validation of signature-related structures (Signatures.xml)
- **writer**: serializing model changes back into OFD package output

## Key Differences from PDF

- OFD is package + XML oriented; PDF is object/stream oriented
- OFD reference resolution is XML/ID driven; PDF relies on object graph mechanics
- OFD signature handling follows OFD document structures; PDF signatures integrate through PDF object structures

## Minimal Phase-1 Milestone (DELIVERED)

- ✅ open an OFD package (ZIP container)
- ✅ read OFD.xml Version attribute and DocRoot
- ✅ parse `Document.xml` and list pages (Page count)
- ✅ validate package structure (OFD.xml, Document.xml presence, DocRoot integrity)
- ✅ extract text from TextCode elements across all pages
- ✅ extract basic metadata (format, path, size, version, page count)

## Phase-1 Non-Goals (Current Limitations)

- full OFD standard coverage (XML model, schema constraints)
- resource mapping and resolution
- signature parsing and validation
- writer/serialization capabilities
- preview rendering and first page inspection
- advanced editing workflows
- complex layout and font handling
