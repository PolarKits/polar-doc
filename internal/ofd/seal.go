package ofd

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// SignatureInfo holds parsed signature metadata from a Signature.xml file.
type SignatureInfo struct {
	ID                 int64
	ProviderName       string
	ProviderVersion    string
	ProviderCompany    string
	SignatureMethod    string
	SignatureDateTime  string
	StampAnnotID       int64
	StampAnnotPageRef  int64
	StampAnnotBoundary []float64
	SealBaseLoc        string
	SignedValuePath    string
}

// SignatureReference represents a single reference in the signature's reference list.
type SignatureReference struct {
	FileRef    string
	CheckValue string
}

// ParseSignaturesXML reads the Signatures.xml file and returns signature entries.
// Returns nil if no signatures are found.
func ParseSignaturesXML(files []*zip.File) ([]SignatureInfo, error) {
	fileIndex := make(map[string]*zip.File, len(files))
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "./")
		fileIndex[name] = f
	}

	signaturesPath := "Doc_0/Signs/Signatures.xml"
	f, ok := fileIndex[signaturesPath]
	if !ok {
		return nil, nil
	}

	data, err := readFileContent(f)
	if err != nil {
		return nil, fmt.Errorf("read Signatures.xml: %w", err)
	}

	return parseSignaturesXMLData(data)
}

func parseSignaturesXMLData(data []byte) ([]SignatureInfo, error) {
	var signatures []SignatureInfo
	decoder := xml.NewDecoder(strings.NewReader(string(data)))

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("xml token: %w", err)
		}

		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		if se.Name.Local == "Signature" {
			sig := SignatureInfo{}
			for _, attr := range se.Attr {
				switch attr.Name.Local {
				case "ID":
					if v, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
						sig.ID = v
					}
				case "BaseLoc":
					sig.SealBaseLoc = attr.Value
				}
			}

			for {
				tok, err := decoder.Token()
				if err != nil {
					break
				}
				if end, ok := tok.(xml.EndElement); ok && end.Name.Local == "Signature" {
					break
				}

				switch v := tok.(type) {
				case xml.StartElement:
					local := v.Name.Local
					if local == "SignedInfo" {
						parseSignedInfo(decoder, &sig, &v)
					}
				}
			}

			if sig.ID != 0 {
				signatures = append(signatures, sig)
			}
		}
	}

	return signatures, nil
}

func parseSignedInfo(decoder *xml.Decoder, sig *SignatureInfo, startEl *xml.StartElement) error {
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}

		if end, ok := tok.(xml.EndElement); ok && end.Name.Local == "SignedInfo" {
			break
		}

		switch v := tok.(type) {
		case xml.StartElement:
			local := v.Name.Local
			switch local {
			case "Provider":
				for _, attr := range v.Attr {
					switch attr.Name.Local {
					case "ProviderName":
						sig.ProviderName = attr.Value
					case "Version":
						sig.ProviderVersion = attr.Value
					case "Company":
						sig.ProviderCompany = attr.Value
					}
				}
			case "SignatureMethod":
				var content string
				if err := decoder.DecodeElement(&content, &v); err == nil {
					sig.SignatureMethod = strings.TrimSpace(content)
				}
			case "SignatureDateTime":
				var content string
				if err := decoder.DecodeElement(&content, &v); err == nil {
					sig.SignatureDateTime = strings.TrimSpace(content)
				}
			case "StampAnnot":
				for _, attr := range v.Attr {
					switch attr.Name.Local {
					case "ID":
						if vv, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
							sig.StampAnnotID = vv
						}
					case "PageRef":
						if vv, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
							sig.StampAnnotPageRef = vv
						}
					case "Boundary":
						sig.StampAnnotBoundary = parseFloatArray(attr.Value)
					}
				}
			case "Seal":
				for {
					tok, err := decoder.Token()
					if err != nil {
						break
					}
					if end, ok := tok.(xml.EndElement); ok && end.Name.Local == "Seal" {
						break
					}
					if start, ok := tok.(xml.StartElement); ok && start.Name.Local == "BaseLoc" {
						var content string
						if err := decoder.DecodeElement(&content, &start); err == nil {
							sig.SealBaseLoc = strings.TrimSpace(content)
						}
					}
				}
			case "SignedValue":
				var content string
				if err := decoder.DecodeElement(&content, &v); err == nil {
					sig.SignedValuePath = strings.TrimSpace(content)
				}
			}
		}
	}

	return nil
}

// SealInfo holds basic metadata parsed from a Seal.esl file header.
// Seal.esl is a binary container format defined in GB/T 33190-2016 §9 that
// embeds an OFD package and a picture (typically PNG) used for visible
// electronic seal rendering.
type SealInfo struct {
	ID       int64
	Version  string
	Width    int64
	Height   int64
	Picture  SealPicture
}

// SealPicture describes the embedded seal picture found within a Seal.esl.
// The picture is typically a PNG image used for visual seal representation.
type SealPicture struct {
	Width        int64
	Height       int64
	Data         []byte
	Format       string
}

// SealParseResults contains the complete result of parsing a Seal.esl file.
type SealParseResults struct {
	Seal   SealInfo
	OFDzip []byte
}

// ParseSealESL reads a Seal.esl binary file and extracts its metadata.
// The Seal.esl format per GB/T 33190-2016 §9 is a cryptographic container that
// embeds an OFD package, X.509 certificates, and a picture (typically PNG).
//
// The container starts with an ASN.1 SEQUENCE header rather than "ES" magic.
// This function detects the actual format and extracts available metadata
// including embedded picture dimensions and data.
//
// Returns SealParseResults on success, or error if the file cannot be parsed.
// This function performs basic structure detection only; it does not perform
// cryptographic signature validation or seal verification.
func ParseSealESL(data []byte) (*SealParseResults, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("seal.esl too short: %d bytes, need ≥8", len(data))
	}

	result := &SealParseResults{}

	// "ES" ASCII header format (2 bytes "ES" + version + big-endian dimensions)
	if data[0] == 0x45 && data[1] == 0x53 {
		result.Seal.Version = fmt.Sprintf("%d.%d", data[2], data[3])
		result.Seal.Width = int64(data[4])<<8 | int64(data[5])
		result.Seal.Height = int64(data[6])<<8 | int64(data[7])
		// ASN.1 SEQUENCE tag (0x30) with 2-byte length encoding (0x82 prefix)
	} else if data[0] == 0x30 && data[1] >= 0x82 {
		result.Seal.Version = "1.0"
	} else {
		return nil, fmt.Errorf("invalid seal format: magic %02x%02x", data[0], data[1])
	}

	if pngData := findPNGInSeal(data); pngData != nil {
		result.Seal.Picture.Format = "PNG"
		result.Seal.Picture.Data = pngData
		if len(pngData) >= 24 {
			// PNG IHDR chunk: bytes 16-19 = width, 20-23 = height (big-endian)
			result.Seal.Picture.Width = int64(pngData[16])<<24 | int64(pngData[17])<<16 | int64(pngData[18])<<8 | int64(pngData[19])
			result.Seal.Picture.Height = int64(pngData[20])<<24 | int64(pngData[21])<<16 | int64(pngData[22])<<8 | int64(pngData[23])
			if result.Seal.Width == 0 {
				result.Seal.Width = result.Seal.Picture.Width
			}
			if result.Seal.Height == 0 {
				result.Seal.Height = result.Seal.Picture.Height
			}
		}
	}

	return result, nil
}

// findPNGEnd locates the end of a PNG chunk sequence starting at data[0].
// Returns the byte index after the last IEND chunk, or 0 if not found.
func findPNGEnd(data []byte) int64 {
	if len(data) < 12 {
		return 0
	}
	if !(data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47) {
		return 0
	}
	pos := int64(8)
	for pos+12 <= int64(len(data)) {
		length := int64(data[pos])<<24 | int64(data[pos+1])<<16 | int64(data[pos+2])<<8 | int64(data[pos+3])
		if pos+12+length > int64(len(data)) {
			return 0
		}
		if data[pos+4] == 0x49 && data[pos+5] == 0x45 && data[pos+6] == 0x4E && data[pos+7] == 0x44 {
			return pos + 12 + length
		}
		pos += 12 + length
	}
	return 0
}

// findPNGInSeal scans binary data for an embedded PNG image.
// Returns the PNG data if found, or nil if not found.
func findPNGInSeal(data []byte) []byte {
	if len(data) < 12 {
		return nil
	}
	for i := 0; i < len(data)-12; i++ {
		if data[i] == 0x89 && data[i+1] == 0x50 && data[i+2] == 0x4E && data[i+3] == 0x47 {
			pngData := data[i:]
			end := findPNGEnd(pngData)
			if end > 0 {
				result := make([]byte, end)
				copy(result, pngData[:end])
				return result
			}
		}
	}
	return nil
}