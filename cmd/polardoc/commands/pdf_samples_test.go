package commands

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
