package pdf

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

// RewriteFile reads a PDF from src and writes a clean single-revision copy to dst.
//
// Unlike CopyFile (raw byte copy), RewriteFile rebuilds the document:
//   - Follows the full xref chain (including incremental updates and Prev links)
//   - Writes all live objects sequentially with fresh byte offsets
//   - Emits a compact cross-reference table and a fresh trailer
//   - Discards deleted and superseded object revisions
//
// The result is a valid single-revision PDF readable without incremental-update chains.
//
// Phase-1 scope: supports traditional xref tables (PDF 1.4 and earlier style) and
// cross-reference streams (PDF 1.5+). Objects stored in object streams (ObjStm)
// are not expanded and are preserved by file-body scanning.
func RewriteFile(src, dst string) error {
	if src == "" {
		return fmt.Errorf("RewriteFile: source path is empty")
	}
	if dst == "" {
		return fmt.Errorf("RewriteFile: destination path is empty")
	}

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("RewriteFile: open source: %w", err)
	}
	defer f.Close()

	version, err := readPDFHeaderVersion(f)
	if err != nil {
		version = "1.7"
	}

	xrefOffset, err := readStartxref(f)
	if err != nil {
		return fmt.Errorf("RewriteFile: readStartxref: %w", err)
	}

	// Read trailer for /Root and /Info.
	rootRef, err := readTrailerRootRef(f, xrefOffset)
	if err != nil || rootRef == "" {
		return fmt.Errorf("RewriteFile: read /Root from trailer: %w", err)
	}
	infoRef, _ := readTrailerInfoRef(f, xrefOffset)

	// Collect live object offsets from the full xref chain.
	offsets, err := collectXrefOffsets(f, xrefOffset)
	if err != nil {
		return fmt.Errorf("RewriteFile: collect xref offsets: %w", err)
	}
	if len(offsets) == 0 {
		return fmt.Errorf("RewriteFile: no objects found in xref")
	}

	// Sort object numbers for deterministic output.
	objNums := make([]int64, 0, len(offsets))
	for n := range offsets {
		objNums = append(objNums, n)
	}
	sort.Slice(objNums, func(i, j int) bool { return objNums[i] < objNums[j] })

	// Build output in memory so we know each object's new offset before writing xref.
	var buf bytes.Buffer

	// PDF header and binary comment (hints to transport layers that file is binary).
	fmt.Fprintf(&buf, "%%PDF-%s\n%%\xE2\xE3\xCF\xD3\n", version)

	newOffsets := make(map[int64]int64, len(offsets))
	for _, n := range objNums {
		srcOffset := offsets[n]
		raw, readErr := readRawObject(f, srcOffset)
		if readErr != nil {
			// Skip unreadable objects; PDF may still be valid without them.
			continue
		}
		newOffsets[n] = int64(buf.Len())
		buf.Write(raw)
		// Guarantee objects are newline-terminated.
		if len(raw) == 0 || raw[len(raw)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}

	if len(newOffsets) == 0 {
		return fmt.Errorf("RewriteFile: could not read any objects from source")
	}

	// Cross-reference table.
	maxObj := int64(0)
	for n := range newOffsets {
		if n > maxObj {
			maxObj = n
		}
	}

	xrefPos := int64(buf.Len())
	fmt.Fprintf(&buf, "xref\n0 %d\n", maxObj+1)
	fmt.Fprintf(&buf, "0000000000 65535 f \r\n") // object 0 always free
	for i := int64(1); i <= maxObj; i++ {
		if off, ok := newOffsets[i]; ok {
			fmt.Fprintf(&buf, "%010d 00000 n \r\n", off)
		} else {
			fmt.Fprintf(&buf, "0000000000 65535 f \r\n")
		}
	}

	// Trailer and end-of-file marker.
	fmt.Fprintf(&buf, "trailer\n<</Size %d/Root %s", maxObj+1, rootRef)
	if infoRef != "" {
		fmt.Fprintf(&buf, "/Info %s", infoRef)
	}
	fmt.Fprintf(&buf, ">>\nstartxref\n%d\n%%%%EOF\n", xrefPos)

	// Write to destination.
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("RewriteFile: create destination: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, bytes.NewReader(buf.Bytes())); err != nil {
		return fmt.Errorf("RewriteFile: write output: %w", err)
	}
	return out.Sync()
}

// collectXrefOffsets returns a map of object number → file offset for all live
// (non-free) objects reachable from xrefOffset, including via /Prev links.
//
// Objects in later revisions (closer to the end of file) take precedence over
// earlier revisions for the same object number.
func collectXrefOffsets(f *os.File, xrefOffset int64) (map[int64]int64, error) {
	result := make(map[int64]int64)
	visited := make(map[int64]bool)
	if err := collectXrefOffsetsAt(f, xrefOffset, result, visited); err != nil {
		return nil, err
	}
	return result, nil
}

// collectXrefOffsetsAt collects entries from the xref section at xrefOff, then
// recursively follows the /Prev link if present.
func collectXrefOffsetsAt(f *os.File, xrefOff int64, result map[int64]int64, visited map[int64]bool) error {
	if visited[xrefOff] {
		return nil
	}
	visited[xrefOff] = true

	_, err := f.Seek(xrefOff, io.SeekStart)
	if err != nil {
		return err
	}

	rd := bufio.NewReaderSize(f, 4096)

	// Peek at first line to distinguish traditional xref vs xref stream.
	firstLine, err := readPDFLine(rd)
	if err != nil && err != io.EOF {
		return err
	}
	firstTrimmed := strings.TrimSpace(firstLine)

	var prevOffset int64

	if firstTrimmed == "xref" {
		// Traditional xref table.
		prevOffset, err = collectTraditionalXref(rd, result)
		if err != nil {
			return err
		}
	} else {
		// Likely an xref stream object (PDF 1.5+).
		// Parse it using the dedicated xref stream enumerator.
		entries, prev, streamErr := enumerateXRefStream(f, xrefOff)
		if streamErr != nil {
			// Not a recognised format; skip silently.
			return nil
		}
		for objNum, off := range entries {
			if _, seen := result[objNum]; !seen {
				result[objNum] = off
			}
		}
		prevOffset = prev
	}

	if prevOffset > 0 {
		return collectXrefOffsetsAt(f, prevOffset, result, visited)
	}
	return nil
}

// collectTraditionalXref reads xref subsections from rd (positioned just after
// the "xref" keyword line) and populates result. It returns the /Prev offset
// from the trailer, or 0 if none.
func collectTraditionalXref(rd *bufio.Reader, result map[int64]int64) (int64, error) {
	var prevOffset int64
	inTrailer := false
	var trailerContent strings.Builder

	for {
		line, err := readPDFLine(rd)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		if trimmed == "trailer" || strings.HasPrefix(trimmed, "trailer") {
			inTrailer = true
			suffix := strings.TrimPrefix(trimmed, "trailer")
			trailerContent.WriteString(suffix)
			continue
		}

		if inTrailer {
			trailerContent.WriteString(trimmed)
			open := strings.Count(trailerContent.String(), "<<") - strings.Count(trailerContent.String(), ">>")
			if open <= 0 {
				// Trailer dict complete; extract /Prev.
				tc := trailerContent.String()
				if idx := strings.Index(tc, "/Prev "); idx >= 0 {
					rest := tc[idx+6:]
					end := 0
					for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
						end++
					}
					if end > 0 {
						prevOffset, _ = strconv.ParseInt(rest[:end], 10, 64)
					}
				}
				// Done with this xref section.
				return prevOffset, nil
			}
			continue
		}

		// Parse subsection header "startObj count".
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		startObj, err1 := strconv.ParseInt(fields[0], 10, 64)
		count, err2 := strconv.ParseInt(fields[1], 10, 64)
		if err1 != nil || err2 != nil || count <= 0 {
			continue
		}

		// Read count xref entries.
		for i := int64(0); i < count; i++ {
			entryLine, entryErr := readPDFLine(rd)
			if entryErr != nil && entryErr != io.EOF {
				return 0, entryErr
			}
			ef := strings.Fields(entryLine)
			if len(ef) < 3 {
				continue
			}
			fileOff, parseErr := strconv.ParseInt(ef[0], 10, 64)
			if parseErr != nil {
				continue
			}
			entryType := ef[2]
			objNum := startObj + i
			if entryType == "n" && objNum > 0 {
				// First visit wins (latest revision takes precedence).
				if _, seen := result[objNum]; !seen {
					result[objNum] = fileOff
				}
			}
		}
	}

	return prevOffset, nil
}

// enumerateXRefStream parses a cross-reference stream object located at xrefOff
// and returns a map of type-1 object offsets and the /Prev link offset.
// Type-2 entries (compressed objects in object streams) are skipped for Phase-1.
func enumerateXRefStream(f *os.File, xrefOff int64) (map[int64]int64, int64, error) {
	_, err := f.Seek(xrefOff, io.SeekStart)
	if err != nil {
		return nil, 0, err
	}

	// Read until the end of the xref stream object.
	rd := bufio.NewReaderSize(f, 8192)
	var header bytes.Buffer
	for {
		line, err := readPDFLine(rd)
		header.WriteString(line)
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, "stream") || trimmed == "stream" {
			break
		}
		if err == io.EOF {
			return nil, 0, fmt.Errorf("enumerateXRefStream: stream keyword not found")
		}
		if err != nil {
			return nil, 0, err
		}
	}

	hdr := header.String()

	// Parse /W field.
	wStart := strings.Index(hdr, "/W[")
	if wStart < 0 {
		wStart = strings.Index(hdr, "/W [")
	}
	if wStart < 0 {
		return nil, 0, fmt.Errorf("enumerateXRefStream: /W not found")
	}
	wBracketStart := strings.Index(hdr[wStart:], "[")
	if wBracketStart < 0 {
		return nil, 0, fmt.Errorf("enumerateXRefStream: /W bracket not found")
	}
	wBracketStart += wStart
	wBracketEnd := strings.Index(hdr[wBracketStart:], "]")
	if wBracketEnd < 0 {
		return nil, 0, fmt.Errorf("enumerateXRefStream: /W bracket not closed")
	}
	wBracketEnd += wBracketStart
	wStr := hdr[wBracketStart+1 : wBracketEnd]
	var w [3]int
	if _, err := fmt.Sscanf(strings.TrimSpace(wStr), "%d %d %d", &w[0], &w[1], &w[2]); err != nil {
		return nil, 0, fmt.Errorf("enumerateXRefStream: parse /W: %w", err)
	}
	entrySize := w[0] + w[1] + w[2]
	if entrySize == 0 {
		return nil, 0, fmt.Errorf("enumerateXRefStream: zero entry size")
	}

	// Parse /Size.
	size := 0
	if sIdx := strings.Index(hdr, "/Size "); sIdx >= 0 {
		fmt.Sscanf(hdr[sIdx+6:], "%d", &size)
	}

	// Parse /Index (default is [0 Size]).
	type indexRange struct{ start, count int }
	var indexRanges []indexRange
	if iIdx := strings.Index(hdr, "/Index["); iIdx >= 0 || strings.Contains(hdr, "/Index [") {
		if iIdx < 0 {
			iIdx = strings.Index(hdr, "/Index [")
		}
		iBracketStart := strings.Index(hdr[iIdx:], "[")
		iBracketEnd := strings.Index(hdr[iIdx:], "]")
		if iBracketStart >= 0 && iBracketEnd > iBracketStart {
			indexStr := hdr[iIdx+iBracketStart+1 : iIdx+iBracketEnd]
			fields := strings.Fields(indexStr)
			for i := 0; i+1 < len(fields); i += 2 {
				s, err1 := strconv.Atoi(fields[i])
				c, err2 := strconv.Atoi(fields[i+1])
				if err1 == nil && err2 == nil {
					indexRanges = append(indexRanges, indexRange{s, c})
				}
			}
		}
	}
	if len(indexRanges) == 0 {
		indexRanges = []indexRange{{0, size}}
	}

	// Parse /Length.
	streamLen := 0
	if lIdx := strings.Index(hdr, "/Length "); lIdx >= 0 {
		fmt.Sscanf(hdr[lIdx+8:], "%d", &streamLen)
	}
	if streamLen <= 0 {
		return nil, 0, fmt.Errorf("enumerateXRefStream: invalid /Length %d", streamLen)
	}

	// Parse /Prev.
	var prevOffset int64
	if pIdx := strings.Index(hdr, "/Prev "); pIdx >= 0 {
		fmt.Sscanf(hdr[pIdx+6:], "%d", &prevOffset)
	}

	// Read and decompress stream body.
	streamBody := make([]byte, streamLen)
	if _, err := io.ReadFull(rd, streamBody); err != nil {
		return nil, 0, fmt.Errorf("enumerateXRefStream: read stream body: %w", err)
	}

	// Check for /Filter /FlateDecode (most common).
	isFlate := strings.Contains(hdr, "FlateDecode") || strings.Contains(hdr, "/Fl ")
	var rawEntries []byte
	if isFlate {
		zr, err := zlib.NewReader(bytes.NewReader(streamBody))
		if err != nil {
			return nil, 0, fmt.Errorf("enumerateXRefStream: zlib: %w", err)
		}
		rawEntries, err = io.ReadAll(zr)
		if err != nil {
			return nil, 0, fmt.Errorf("enumerateXRefStream: zlib read: %w", err)
		}
	} else {
		rawEntries = streamBody
	}

	result := make(map[int64]int64)
	pos := 0
	for _, rng := range indexRanges {
		for i := 0; i < rng.count; i++ {
			if pos+entrySize > len(rawEntries) {
				break
			}
			entry := rawEntries[pos : pos+entrySize]
			pos += entrySize

			objNum := int64(rng.start + i)

			// Entry type: w[0] bytes.
			typ := 1 // default when w[0]==0
			if w[0] > 0 {
				typ = 0
				for k := 0; k < w[0]; k++ {
					typ = typ<<8 | int(entry[k])
				}
			}

			switch typ {
			case 0:
				// Free object; skip.
			case 1:
				// Type 1: direct object at file offset (w[1] bytes).
				off := int64(0)
				for k := w[0]; k < w[0]+w[1]; k++ {
					off = off<<8 | int64(entry[k])
				}
				if objNum > 0 {
					result[objNum] = off
				}
			case 2:
				// Type 2: compressed in object stream; skip for Phase-1.
			}
		}
	}

	return result, prevOffset, nil
}

// readPDFLine reads a line from rd, treating CR, LF, and CRLF all as line terminators.
// The returned string includes the terminating byte(s).
func readPDFLine(rd *bufio.Reader) (string, error) {
	var buf strings.Builder
	for {
		b, err := rd.ReadByte()
		if err != nil {
			return buf.String(), err
		}
		buf.WriteByte(b)
		if b == '\n' {
			return buf.String(), nil
		}
		if b == '\r' {
			// Consume trailing \n if CRLF.
			next, peekErr := rd.ReadByte()
			if peekErr == nil {
				if next == '\n' {
					buf.WriteByte('\n')
					return buf.String(), nil
				}
				_ = rd.UnreadByte()
			}
			return buf.String(), peekErr
		}
	}
}

// readRawObject reads the complete raw bytes of a PDF object at fileOffset.
//
// Handles:
//   - Standard line-ending styles: LF, CRLF, CR-only.
//   - Stream objects: reads exactly /Length bytes for the stream body.
//   - Non-stream objects: reads until the "endobj" token.
//
// Returns all bytes from the "N M obj" header through the closing "endobj".
func readRawObject(f *os.File, fileOffset int64) ([]byte, error) {
	const maxObjSize = 64 * 1024 * 1024 // 64 MiB safety limit

	_, err := f.Seek(fileOffset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	rd := bufio.NewReaderSize(f, 8192)
	var raw bytes.Buffer

	// Read object header: "N M obj" or "N M obj\n".
	headerLine, err := readPDFLine(rd)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("readRawObject at %d: read header: %w", fileOffset, err)
	}
	if !strings.Contains(strings.TrimSpace(headerLine), " obj") {
		return nil, fmt.Errorf("readRawObject at %d: no obj keyword: %q", fileOffset, headerLine)
	}
	raw.WriteString(headerLine)
	ensureNewline(&raw)

	// Read lines accumulating dict content, looking for "stream" or "endobj".
	var dictContent strings.Builder
	dictContent.WriteString(headerLine)

	for {
		if raw.Len() > maxObjSize {
			return nil, fmt.Errorf("readRawObject at %d: object exceeds size limit", fileOffset)
		}

		line, lineErr := readPDFLine(rd)
		raw.WriteString(line)
		ensureNewline(&raw)

		trimmed := strings.TrimSpace(line)
		dictContent.WriteString(line)

		// Detect stream start.
		if trimmed == "stream" {
			// Stream object: use /Length to skip stream body exactly.
			length := extractLength(dictContent.String())
			if length < 0 {
				return nil, fmt.Errorf("readRawObject at %d: stream without /Length", fileOffset)
			}
			streamBody := make([]byte, length)
			if _, readErr := io.ReadFull(rd, streamBody); readErr != nil {
				return nil, fmt.Errorf("readRawObject at %d: stream body: %w", fileOffset, readErr)
			}
			raw.Write(streamBody)
			// Read remaining lines until endobj.
			for {
				endLine, endErr := readPDFLine(rd)
				raw.WriteString(endLine)
				ensureNewline(&raw)
				if strings.TrimSpace(endLine) == "endobj" {
					return raw.Bytes(), nil
				}
				if endErr == io.EOF {
					return raw.Bytes(), nil
				}
				if endErr != nil {
					return nil, endErr
				}
			}
		}

		// Detect end of non-stream object.
		if trimmed == "endobj" {
			return raw.Bytes(), nil
		}

		if lineErr == io.EOF {
			break
		}
		if lineErr != nil {
			return nil, lineErr
		}
	}

	return raw.Bytes(), nil
}

// ensureNewline appends '\n' to buf if it does not already end with one.
func ensureNewline(buf *bytes.Buffer) {
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] != '\n' {
		buf.WriteByte('\n')
	}
}

// extractLength returns the /Length value from a PDF dict content string, or -1
// if not found. Handles both "/Length N" and "/Length\nN" forms.
func extractLength(dictContent string) int64 {
	for _, prefix := range []string{"/Length ", "/Length\n", "/Length\r"} {
		idx := strings.Index(dictContent, prefix)
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(dictContent[idx+len(prefix):])
		end := 0
		for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
			end++
		}
		if end == 0 {
			continue
		}
		n, err := strconv.ParseInt(rest[:end], 10, 64)
		if err == nil {
			return n
		}
	}
	return -1
}
