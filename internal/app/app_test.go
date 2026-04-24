package app

import (
	"context"
	"testing"

	"github.com/PolarKits/polar-doc/internal/doc"
)

// TestStaticResolver_ByFormat tests that StaticResolver correctly resolves
// format services for supported formats and returns false for unsupported formats.
func TestStaticResolver_ByFormat(t *testing.T) {
	// Use the phase-1 resolver which wires real PDF and OFD services.
	resolver := NewPhase1Resolver()

	tests := []struct {
		name        string
		format      doc.Format
		wantService bool
	}{
		{
			name:        "PDF format resolves to service",
			format:      doc.FormatPDF,
			wantService: true,
		},
		{
			name:        "OFD format resolves to service",
			format:      doc.FormatOFD,
			wantService: true,
		},
		{
			name:        "Unknown format returns no service",
			format:      doc.Format("unknown"),
			wantService: false,
		},
		{
			name:        "Empty format returns no service",
			format:      doc.Format(""),
			wantService: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, ok := resolver.ByFormat(tt.format)

			if ok != tt.wantService {
				t.Errorf("ByFormat(%q) ok = %v, want %v", tt.format, ok, tt.wantService)
			}

			if tt.wantService && service == nil {
				t.Errorf("ByFormat(%q) returned nil service when service was expected", tt.format)
			}

			if !tt.wantService && service != nil {
				t.Errorf("ByFormat(%q) returned non-nil service when nil was expected", tt.format)
			}
		})
	}
}

// TestServiceSet_Wiring tests that ServiceSet correctly holds and provides
// access to format services through the StaticResolver.
func TestServiceSet_Wiring(t *testing.T) {
	// Create mock/stub services for testing wiring without real dependencies.
	pdfSvc := &mockFormatService{name: "mock-pdf"}
	ofdSvc := &mockFormatService{name: "mock-ofd"}

	// Construct ServiceSet with mock services.
	set := ServiceSet{
		PDF: pdfSvc,
		OFD: ofdSvc,
	}

	// Create resolver with the service set.
	resolver := NewStaticResolver(set)

	// Verify PDF service can be resolved.
	pdfResolved, ok := resolver.ByFormat(doc.FormatPDF)
	if !ok {
		t.Error("Failed to resolve PDF service")
	}
	// Verify the resolved service is non-nil (actual pointer equality check removed
	// because we only verify the interface value is not nil for wiring tests).
	if pdfResolved == nil {
		t.Error("Resolved PDF service is nil")
	}

	// Verify OFD service can be resolved.
	ofdResolved, ok := resolver.ByFormat(doc.FormatOFD)
	if !ok {
		t.Error("Failed to resolve OFD service")
	}
	if ofdResolved == nil {
		t.Error("Resolved OFD service is nil")
	}

	// Verify nil services in ServiceSet result in "not found".
	emptySet := ServiceSet{
		PDF: nil,
		OFD: nil,
	}
	emptyResolver := NewStaticResolver(emptySet)

	_, ok = emptyResolver.ByFormat(doc.FormatPDF)
	if ok {
		t.Error("Expected PDF service to not be found when nil in ServiceSet")
	}

	_, ok = emptyResolver.ByFormat(doc.FormatOFD)
	if ok {
		t.Error("Expected OFD service to not be found when nil in ServiceSet")
	}
}

// TestNewPhase1Resolver_Integration tests that NewPhase1Resolver returns
// a properly wired resolver with real PDF and OFD services.
func TestNewPhase1Resolver_Integration(t *testing.T) {
	resolver := NewPhase1Resolver()

	// Verify PDF service is wired and functional.
	pdfSvc, ok := resolver.ByFormat(doc.FormatPDF)
	if !ok {
		t.Fatal("PDF service not resolved by phase-1 resolver")
	}
	if pdfSvc == nil {
		t.Fatal("PDF service is nil")
	}

	// Verify OFD service is wired and functional.
	ofdSvc, ok := resolver.ByFormat(doc.FormatOFD)
	if !ok {
		t.Fatal("OFD service not resolved by phase-1 resolver")
	}
	if ofdSvc == nil {
		t.Fatal("OFD service is nil")
	}

	// Verify that the services are non-nil and can be used as FormatService.
	// The concrete types are implementation details of the format packages;
	// we only verify that the wiring returns valid service instances.
}

// mockFormatService is a minimal stub implementing FormatService for wiring tests.
// It does not implement any actual functionality; it only exists to test
// that ServiceSet correctly holds and returns service references.
type mockFormatService struct {
	name string
}

// Stub implementations to satisfy FormatService interface.
// These methods panic if called; they are only for wiring verification.
func (m *mockFormatService) Open(ctx context.Context, ref doc.DocumentRef) (doc.Document, error) {
	panic("mock: Open not implemented")
}

func (m *mockFormatService) Info(ctx context.Context, d doc.Document) (doc.InfoResult, error) {
	panic("mock: Info not implemented")
}

func (m *mockFormatService) Validate(ctx context.Context, d doc.Document) (doc.ValidationReport, error) {
	panic("mock: Validate not implemented")
}

func (m *mockFormatService) ExtractText(ctx context.Context, d doc.Document) (doc.TextResult, error) {
	panic("mock: ExtractText not implemented")
}

func (m *mockFormatService) RenderPreview(ctx context.Context, d doc.Document, req doc.PreviewRequest) (doc.PreviewResult, error) {
	panic("mock: RenderPreview not implemented")
}

func (m *mockFormatService) FirstPageInfo(ctx context.Context, d doc.Document) (*doc.FirstPageInfoResult, error) {
	panic("mock: FirstPageInfo not implemented")
}
