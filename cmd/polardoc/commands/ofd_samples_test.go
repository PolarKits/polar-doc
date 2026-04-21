package commands

import (
	"testing"

	fixtures "github.com/PolarKits/polardoc/internal/testdata"
)

func requireOFDSample(t *testing.T, key string) string {
	t.Helper()

	sample, ok := fixtures.OFDSampleByKey(key)
	if !ok {
		t.Fatalf("missing OFD sample %q", key)
	}
	return sample.Path()
}
