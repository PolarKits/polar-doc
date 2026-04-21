package testdata

import "path/filepath"

// OFDSample describes a single OFD fixture file.
type OFDSample struct {
	// Key is the unique identifier for this sample (used in tests and lookups).
	Key string
	// Filename is the actual file name in testdata/ofd/.
	Filename string
	// Category groups samples by purpose (core, feature, etc.).
	Category string
	// Description explains what this sample is meant to test.
	Description string
	// ExpectText is true when the fixture is expected to yield non-empty extracted text.
	ExpectText bool
}

// Path returns the absolute path to the fixture file.
func (s OFDSample) Path() string {
	return filepath.Join(repoRoot(), "testdata", "ofd", s.Filename)
}

// OFDSamples returns all registered OFD fixtures.
func OFDSamples() []OFDSample {
	return append([]OFDSample(nil), ofdSamples...)
}

// OFDSampleByKey returns the OFD fixture with the given key, or (zero, false).
func OFDSampleByKey(key string) (OFDSample, bool) {
	for _, sample := range ofdSamples {
		if sample.Key == key {
			return sample, true
		}
	}
	return OFDSample{}, false
}

var ofdSamples = []OFDSample{
	{Key: "core-helloworld", Filename: "test_core_helloworld.ofd", Category: "core", Description: "Minimal hello-world OFD fixture", ExpectText: true},
	{Key: "core-multipage", Filename: "test_core_multipage.ofd", Category: "core", Description: "Multi-page OFD fixture", ExpectText: true},
	{Key: "feature-keyword-search", Filename: "test_feat_keyword_search.ofd", Category: "feature", Description: "OFD fixture with searchable keywords", ExpectText: true},
	{Key: "feature-invoice", Filename: "test_feat_invoice.ofd", Category: "feature", Description: "Invoice OFD fixture", ExpectText: true},
	{Key: "feature-complex-layout", Filename: "test_feat_complex_layout.ofd", Category: "feature", Description: "Complex layout OFD fixture", ExpectText: false},
	{Key: "feature-images", Filename: "test_feat_images.ofd", Category: "feature", Description: "Image-heavy OFD fixture", ExpectText: false},
	{Key: "feature-attachment", Filename: "test_feat_attachment.ofd", Category: "feature", Description: "OFD fixture with embedded attachments", ExpectText: false},
	{Key: "feature-signature", Filename: "test_feat_signature.ofd", Category: "feature", Description: "OFD fixture with digital signature", ExpectText: false},
	{Key: "feature-transparency", Filename: "test_feat_transparency.ofd", Category: "feature", Description: "OFD fixture with transparency", ExpectText: false},
	{Key: "feature-pattern", Filename: "test_feat_pattern.ofd", Category: "feature", Description: "OFD fixture with pattern fill", ExpectText: false},
}
