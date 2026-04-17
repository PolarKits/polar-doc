# OFD Domain

## Purpose

The OFD domain provides OFD-specific container, XML, signature, validation, and writing behavior for PolarDoc.
It keeps OFD semantics explicit and separate from PDF internals.

## Responsibilities

- open and manage OFD package contents
- parse OFD XML structures into OFD-native models
- resolve IDs and references across document parts
- process signature-related structures
- validate package, XML, and reference consistency
- write valid OFD package output

## Core Components

- container:
  - reads package entries and normalized paths
  - provides deterministic access to OFD resources
- XML model:
  - maps core XML structures into internal types
  - preserves schema intent for operations and validation
- resolver:
  - resolves references across document, page, and resource scopes
  - reports missing or ambiguous links explicitly
- signature:
  - parses and validates signature-related structures
  - returns explicit signature status to callers
- writer:
  - serializes model changes back into OFD package output
  - keeps package layout stable where practical
- validator:
  - checks schema constraints, reference integrity, and container consistency
  - produces structured findings

## Key Differences from PDF

- OFD is package + XML oriented; PDF is object/stream oriented
- OFD reference resolution is XML/ID driven; PDF relies on object graph mechanics
- OFD signature handling follows OFD document structures; PDF signatures integrate through PDF object structures

## Minimal Phase-1 Milestone

- open an OFD package
- parse `Document.xml`
- list pages
- extract basic metadata

## Phase-1 Non-Goals

- full OFD standard coverage
- advanced editing workflows
- advanced rendering pipelines
