package commands

import (
	"testing"

	fixtures "github.com/PolarKits/polar-doc/internal/testdata"
)

func requireOFDSample(t *testing.T, key string) string {
	t.Helper()

	sample, ok := fixtures.OFDSampleByKey(key)
	if !ok {
		t.Fatalf("missing OFD sample %q", key)
	}
	return sample.Path()
}
