// Package security contains cryptographic, signature, and trust-related concerns.
//
// This package defines the signing contract (SignService interface) for
// format-specific document signatures.
//
// # Phase-1 Scope
//
// Phase-1: signing is not implemented. SignService is defined as a future
// extension point. No cryptographic operations, certificate handling, or
// signature verification exist in phase-1.
//
// Standards mapping:
//   - PDF signatures: ISO 32000-2:2020 §12.8 (digital signatures)
//   - OFD signatures: GB/T 33190-2016 §9 (digital signatures)
//
// These are future capabilities requiring significant additional infrastructure
// (crypto providers, certificate chains, timestamp services, etc.).
package security
