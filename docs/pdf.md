# PDF Domain

## Purpose

The PDF domain provides PDF-specific parsing, validation, repair, and writing for PolarDoc.
It keeps PDF behavior explicit and isolated from OFD.

## Responsibilities

- parse PDF bytes into PDF-native structures
- hold a PDF-focused internal model for domain operations
- write valid PDF output from domain state
- validate structure and consistency rules
- apply bounded repair for common structural damage

## Component Separation

- parser:
  - reads header, xref/trailer, objects, and streams
  - reports parse diagnostics with location and severity
- model:
  - stores PDF-native structures used by operations
  - does not attempt to represent OFD concepts
- writer:
  - emits valid PDF output
  - supports full rewrite first, then incremental update paths
- validator:
  - checks structural integrity and consistency constraints
  - returns structured findings for callers
- repair:
  - performs explicit, bounded recovery steps
  - never masks unrecoverable corruption

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

## Minimal Phase-1 Milestone

- open a PDF file
- inspect core structure (header, trailer, xref, objects)
- extract basic text from common text objects
- rewrite to a clean, valid PDF output

## Phase-1 Non-Goals

- full PDF specification coverage
- visual editing workflows
