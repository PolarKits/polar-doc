# PDF Compatibility Design

PolarDoc PDF compatibility strategy covering ISO 32000-1 (PDF 1.7) and ISO 32000-2 (PDF 2.0).

## Version Landscape

| Version | Year | Key Features | Distribution |
|---------|------|--------------|---------------|
| 1.0–1.3 | 1993–1999 | Basic structure, RC4-40 encryption, signatures | Rare |
| 1.4 | 2001 | Transparency model, Optional Content, tagged PDF | Large legacy base |
| 1.5 | 2003 | Cross-reference streams, Object streams (★ major break) | Large legacy base |
| 1.6 | 2004 | AES-128, 3D U3D | Small base |
| 1.7 | 2006 | AES-256 extension, ISO 32000-1:2008 | Current mainstream (★ baseline) |
| 2.0 | 2017 | ISO 32000-2:2017, deprecated XFA/Info dict, page-level output intents | Growing (★ modern target) |

### PDF/A Archival Standards (write-relevant)

| Standard | Base Version | Notes |
|----------|-------------|-------|
| PDF/A-1 | PDF 1.4 | No transparency, no encryption |
| PDF/A-2 | PDF 1.7 | Allows transparency, JPEG2000 |
| PDF/A-3 | PDF 1.7 | Allows arbitrary file attachments |
| PDF/A-4 | PDF 2.0 | Modern archival target |

## Read vs Write Strategy

**Principle: Read permissively (tolerate all versions), Write strictly (output 1.7 or 2.0 only).**

| Version | Read | Write | Reason |
|---------|------|-------|--------|
| 1.0–1.3 | ✓ | ✗ | Read-only: too old to generate |
| 1.4 | ✓ | ✗ | Read-only: large legacy base |
| 1.5 | ✓ | ✗ | Read-only: object streams / xref streams must be readable |
| 1.6 | ✓ | ✗ | Read-only: AES-128 must be decryptable |
| 1.7 | ✓ | ✓ | ISO 32000-1, maximum compatibility baseline |
| 2.0 | ✓ | ✓ | ISO 32000-2, preferred modern target |

Downgrading output below PDF 1.7 is not allowed.
Downgrading PDF 2.0 input to PDF 1.7 output returns `ErrDowngradeNotAllowed`.

## Feature Detection Over Version Gating

Do not gate parsing paths on the `%PDF-X.Y` header alone. Probe actual document structures:
- xref: detect xref stream vs traditional table at startxref offset
- object streams: detect ObjStm entries in xref index
- encryption: check /Encrypt dict before any read
- metadata: try XMP stream first (PDF 2.0 primary), fall back to Info dict

`PDFFeatureSet` captures the effective feature profile of an opened document.

## XRef Parsing Architecture

```
readStartxref()
    ↓ xrefOffset
    ├── starts with "xref"         → parseTraditionalXRef()  (1.0+)
    ├── starts with "N G obj <<"   → parseXRefStream()        (1.5+)
    └── both present               → isHybridXRef = true; prefer xref stream (ISO 32000-1 §C.2)

parseXRefStream():
    read stream dict: /Type/XRef /W /Index /Filter
    decompress (typically FlateDecode)
    parse binary records using /W field widths
    record types:
        type=0: free object
        type=1: uncompressed object (field2=offset, field3=generation)
        type=2: compressed object in ObjStm (field2=stm_obj_num, field3=index_in_stm)

parseObjectStream(stmObjNum):
    read ObjStm object (dict has /N count, /First body-start offset)
    decompress stream
    parse N pairs of (obj_num, offset) as index
    read object bodies at those offsets
```

## PDF Association Fix Issues

Known spec ambiguities and tool bugs handled as silent fixes with warning records:

| Fix | Description | Common Cause |
|-----|-------------|-------------|
| FixHybridXRef | Both xref table and xref stream present; prefer xref stream | Early Acrobat bug |
| FixBrokenStartxref | startxref offset invalid; scan backward from EOF | Buggy third-party writers |
| FixMissingEOF | File ends without %%EOF marker | Server-generated PDFs |
| FixInfoDictUTF16NoBOM | Info dict strings in UTF-16BE without BOM | Acrobat < 6.0 |
| FixStreamLengthMismatch | /Length wrong; use endstream keyword as boundary | Post-edit length drift |
| FixTrailerPrevChain | Prev chain loops or points to invalid offset; stop recursion | Incremental update bugs |
| FixNullObjectRef | "0 0 R" treated as null rather than error | Incorrect indirect ref usage |
| FixEmptyEncryptDict | /Encrypt key exists but dict is empty; treat as unencrypted | Partial encryption removal |

`DefaultCompatFixes` enables all of the above.

## Save Strategy

| Strategy | Description | Use Case |
|----------|-------------|----------|
| `StrategyPassthrough` | Raw byte copy, no parsing | Unmodified file transfer |
| `StrategyIncrementalAppend` | Append new revision; preserves original bytes and signatures | Metadata / annotation edits |
| `StrategyFullRewrite` | Rebuild entire document at target version | Structural changes, archival |

## Default Save Policy

| Scenario | Strategy | Target Version | Reason |
|----------|----------|----------------|--------|
| Unmodified transfer | Passthrough | — | Byte-identical, zero risk |
| Metadata / annotation edit | IncrementalAppend | PDF 1.7 | Preserves signatures, max compat |
| Structural modification | FullRewrite | PDF 1.7 | Clean output, no revision baggage |
| Long-term archival | FullRewrite | PDF 2.0 | ISO 32000-2, XMP metadata |
| Regulatory delivery | FullRewrite | PDF/A-3 (1.7) or PDF/A-4 (2.0) | Compliance requirement |

## Version Upgrade Rules

Upgrade is performed during FullRewrite only.

| Input Version | → PDF 1.7 | → PDF 2.0 |
|---------------|-----------|-----------|
| 1.0–1.3 | Update header; rebuild xref table; preserve objects | + XMP metadata; AES-256 if encrypted |
| 1.4 | Update header; transparency already compatible | Same + MigrateInfoDictToXMP |
| 1.5–1.6 | Update header; preserve xref stream/ObjStm; AES-128→256 optional | + MigrateInfoDictToXMP; RemoveXFA |
| 1.7 | No-op (already target) | MigrateInfoDictToXMP; RemoveXFA; AES-256 upgrade |
| 2.0 | Downgrade not allowed | No-op (already target) |

## Phase-2 Implementation Priority

1. **XRef reading** — `parseXRefStream`, `parseObjectStream`, `CompatReader.ResolveObject`
2. **Write pipeline** — `StrategyIncrementalAppend`, `StrategyFullRewrite` at PDF 1.7
3. **Version upgrade** — 1.4–1.6 → 1.7, Info dict → XMP, 1.7 → 2.0
4. **Encryption** (Phase-3) — AES-256 read decrypt, RC4 compat, encrypted write
