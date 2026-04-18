// Package app wires top-level application flows for CLI, MCP, and future services.
//
// This is a pure routing and composition layer. It does not implement any document
// format semantics (PDF, OFD, or otherwise) and does not reference any standard clause.
// The application layer depends on doc contracts, delegates to format services via
// ServiceResolver, and remains agnostic to internal format details.
//
// # Phase-1 Scope
//
// In phase-1, this package assembles the CLI resolver (NewPhase1Resolver) and
// provides ServiceResolver dispatch. It contains no new capability definitions.
package app
