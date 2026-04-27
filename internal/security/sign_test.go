package security

import (
	"context"
	"strings"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
)

func TestNewPDFSigner(t *testing.T) {
	signer := NewPDFSigner()
	if signer == nil {
		t.Fatal("NewPDFSigner returned nil")
	}
}

func TestNewOFDSigner(t *testing.T) {
	signer := NewOFDSigner()
	if signer == nil {
		t.Fatal("NewOFDSigner returned nil")
	}
}

func TestSignService_PDFSigner_Sign(t *testing.T) {
	signer := NewPDFSigner()
	_, err := signer.Sign(context.Background(), &mockDoc{}, doc.SignRequest{})
	if err == nil {
		t.Fatal("Sign: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "signing is not implemented for pdf") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "signing is not implemented for pdf")
	}
}

func TestSignService_OFDSigner_Sign(t *testing.T) {
	signer := NewOFDSigner()
	_, err := signer.Sign(context.Background(), &mockDoc{}, doc.SignRequest{})
	if err == nil {
		t.Fatal("Sign: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "signing is not implemented for ofd") {
		t.Fatalf("error = %q, want contains %q", err.Error(), "signing is not implemented for ofd")
	}
}

func TestSignService_Interface(t *testing.T) {
	var _ SignService = NewPDFSigner()
	var _ SignService = NewOFDSigner()
}

type mockDoc struct{}

func (m *mockDoc) Ref() doc.DocumentRef {
	return doc.DocumentRef{Format: doc.FormatPDF, Path: "/nonexistent.pdf"}
}

func (m *mockDoc) Close() error {
	return nil
}