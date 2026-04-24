package mcp

import (
	"testing"

	fixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

func requirePDFSample(t *testing.T, key string) string {
	t.Helper()

	sample, ok := fixtures.PDFSampleByKey(key)
	if !ok {
		t.Fatalf("missing PDF sample %q", key)
	}
	return sample.Path()
}

func requireOFDSample(t *testing.T, key string) string {
	t.Helper()

	sample, ok := fixtures.OFDSampleByKey(key)
	if !ok {
		t.Fatalf("missing OFD sample %q", key)
	}
	return sample.Path()
}
