package pdf

import (
	"testing"
)

func TestParseXMPMetadata_Basic(t *testing.T) {
	xmp := []byte(`<?xml version="1.0"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/"
                    xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
                    xmlns:xmp="http://ns.adobe.com/xap/1.0/">
      <dc:title>Test Document Title</dc:title>
      <dc:creator>John Doe</dc:creator>
      <pdf:Producer>Adobe Acrobat</pdf:Producer>
      <xmp:CreatorTool>My Application</xmp:CreatorTool>
      <xmp:CreateDate>2024-01-15T10:30:00Z</xmp:CreateDate>
      <xmp:ModifyDate>2024-06-20T15:45:00Z</xmp:ModifyDate>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>`)

	result := parseXMPMetadata(xmp)

	if result["title"] != "Test Document Title" {
		t.Errorf("title = %q, want %q", result["title"], "Test Document Title")
	}
	if result["creator"] != "John Doe" {
		t.Errorf("creator = %q, want %q", result["creator"], "John Doe")
	}
	if result["producer"] != "Adobe Acrobat" {
		t.Errorf("producer = %q, want %q", result["producer"], "Adobe Acrobat")
	}
}

func TestParseXMPMetadata_Empty(t *testing.T) {
	result := parseXMPMetadata(nil)
	if len(result) != 0 {
		t.Errorf("expected empty result for nil input, got %v", result)
	}

	result = parseXMPMetadata([]byte{})
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %v", result)
	}
}

func TestParseXMPMetadata_NoMatchingTags(t *testing.T) {
	xmp := []byte(`<?xml version="1.0"?><root><other>value</other></root>`)
	result := parseXMPMetadata(xmp)
	if len(result) != 0 {
		t.Errorf("expected empty result, got %v", result)
	}
}

func TestParseXMPMetadata_PartialFields(t *testing.T) {
	xmp := []byte(`<?xml version="1.0"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/">
      <dc:title>Partial Doc</dc:title>
    </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>`)

	result := parseXMPMetadata(xmp)
	if result["title"] != "Partial Doc" {
		t.Errorf("title = %q, want %q", result["title"], "Partial Doc")
	}
	if _, ok := result["creator"]; ok {
		t.Error("expected no creator, got one")
	}
}

func TestParseXMPMetadata_WithRDFList(t *testing.T) {
	xmp := []byte(`<?xml version="1.0"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:dc="http://purl.org/dc/elements/1.1/">
  <rdf:Description>
    <dc:title>Document Title</dc:title>
    <dc:creator>Author One</dc:creator>
  </rdf:Description>
</rdf:RDF>`)

	result := parseXMPMetadata(xmp)
	if result["title"] != "Document Title" {
		t.Errorf("title = %q, want %q", result["title"], "Document Title")
	}
	if result["creator"] != "Author One" {
		t.Errorf("creator = %q, want %q", result["creator"], "Author One")
	}
}

func TestParseXMPMetadata_WithSimpleCDATA(t *testing.T) {
	xmp := []byte(`<dc:title>Special Characters: test</dc:title>`)

	result := parseXMPMetadata(xmp)
	if result["title"] != "Special Characters: test" {
		t.Errorf("title = %q, want %q", result["title"], "Special Characters: test")
	}
}

func TestExtractXMLValueSimple(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<tag>value</tag>", "value"},
		{"<tag><nested>value</nested></tag>", "value"},
		{"  <tag>  value  </tag>  ", "value"},
		{"plain text", "plain text"},
		{"<tag>value", "value"},
	}

	for _, tt := range tests {
		got := extractXMLValueSimple(tt.input)
		if got != tt.want {
			t.Errorf("extractXMLValueSimple(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}