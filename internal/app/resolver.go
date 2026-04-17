package app

import (
	"github.com/PolarKits/polardoc/internal/doc"
	"github.com/PolarKits/polardoc/internal/ofd"
	"github.com/PolarKits/polardoc/internal/pdf"
)

// StaticResolver resolves services from a fixed set.
type StaticResolver struct {
	services ServiceSet
}

// NewStaticResolver creates a simple format resolver.
func NewStaticResolver(services ServiceSet) *StaticResolver {
	return &StaticResolver{services: services}
}

// NewPhase1Resolver wires format services for phase-1 CLI flow.
func NewPhase1Resolver() ServiceResolver {
	return NewStaticResolver(ServiceSet{
		PDF: pdf.NewService(),
		OFD: ofd.NewService(),
	})
}

// ByFormat resolves a format service without reflection or plugins.
func (r *StaticResolver) ByFormat(format doc.Format) (FormatService, bool) {
	switch format {
	case doc.FormatPDF:
		return r.services.PDF, r.services.PDF != nil
	case doc.FormatOFD:
		return r.services.OFD, r.services.OFD != nil
	default:
		return nil, false
	}
}
