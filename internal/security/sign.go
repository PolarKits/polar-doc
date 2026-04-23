package security

import (
	"context"
	"fmt"

	"github.com/PolarKits/polar-doc/internal/doc"
)

type pdfSigner struct{}
type ofdSigner struct{}

func (s *pdfSigner) Sign(_ context.Context, d doc.Document, _ doc.SignRequest) (doc.SignResult, error) {
	return doc.SignResult{}, fmt.Errorf("signing is not implemented for pdf")
}

func (s *ofdSigner) Sign(_ context.Context, d doc.Document, _ doc.SignRequest) (doc.SignResult, error) {
	return doc.SignResult{}, fmt.Errorf("signing is not implemented for ofd")
}

// NewPDFSigner returns a SignService that signs PDF documents.
//
// Phase-1 implementation: this is a stub that returns an error indicating
// signing is not implemented. Real PDF signing requires a crypto provider,
// certificate chain validation, and key material, all planned for Phase-2.
// See ISO 32000-2:2020 §12.8 for the PDF digital signature standard.
func NewPDFSigner() SignService {
	return &pdfSigner{}
}

// NewOFDSigner returns a SignService that signs OFD documents.
//
// Phase-1 implementation: this is a stub that returns an error indicating
// signing is not implemented. Real OFD signing requires a crypto provider,
// certificate chain validation, and key material, all planned for Phase-2.
// See GB/T 33190-2016 §9 for the OFD digital signature standard.
func NewOFDSigner() SignService {
	return &ofdSigner{}
}
