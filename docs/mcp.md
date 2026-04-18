# MCP

## MCP Purpose

The PolarDoc MCP server exposes document capabilities through a safe protocol surface.
It provides controlled execution instead of raw file mutation.

## Tools Concept

Tools are capability-oriented and explicit.
Typical tools include document info, validation, extraction, preview write, and commit write.
Each tool defines clear input, output, and error contracts.

## Read vs Write Separation

Read and write operations are separated by design.

- read tools inspect, validate, extract, and preview
- write tools apply approved changes through controlled operations

Read tools are side-effect free.
Write tools require explicit commit intent.

## Preview Before Commit

All write flows are two-step:

1. preview planned changes, affected targets, and risk
2. commit the approved plan without hidden modifications

Preview never mutates files.

## Safety Constraints

- no raw file mutation tools exposed to MCP clients
- no arbitrary path writes outside allowed scope
- no cross-format shortcut that breaks PDF/OFD semantics
- validate preconditions before commit
- return structured errors when checks fail

## Available Tools

### pdf_first_page_info

Extracts structured first page information from a PDF document.

**Input:**
```json
{
  "path": "/path/to/document.pdf"
}
```

**Output (success):**
```json
{
  "path": "/path/to/document.pdf",
  "pages_ref": {"obj_num": 5, "gen_num": 0},
  "page_ref": {"obj_num": 18, "gen_num": 0},
  "parent": {"obj_num": 5, "gen_num": 0},
  "media_box": [0, 0, 612, 792],
  "resources": {"obj_num": 0, "gen_num": 0},
  "contents": [{"obj_num": 19, "gen_num": 0}],
  "rotate": 0
}
```

**Output (error):**
```json
{
  "error": "error message"
}
```

**Supported formats:** PDF only. OFD returns error.

**Known limitations:**
- Only returns first page information
- Inline resources in PDF show `resources.obj_num: 0`
- Corrupted XRef PDFs return specific error messages

## Compatibility Matrix

### `pdf_first_page_info` — testdata/pdf samples

| Sample | Result | Notes |
|--------|--------|-------|
| `pdf20-utf8-test.pdf` | ✓ Success | PDF 2.0, UTF-8 text |
| `Red_Hat_OpenShift_Serverless-1.35-Serverless_Logic-en-US.pdf` | ✓ Success | Commercial document |
| `sample-local-pdf.pdf` | ✓ Success | Local sample |
| `testPDF_Version.5.x.pdf` | ✓ Success | PDF 1.5 |
| `testPDF_Version.8.x.pdf` | ✗ Error | Known corrupted XRef |

### `testPDF_Version.8.x.pdf` failure semantics

When `pdf_first_page_info` is called on this file, it returns:

```json
{
  "error": "first page info: ReadFirstPageInfo: find object 14 offset: object 14 not found in xref"
}
```

This is expected behavior — the XRef table is damaged and the parser correctly reports the failure rather than returning incomplete data.
