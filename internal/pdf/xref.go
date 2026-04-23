package pdf

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// xrefEntryKind identifies how a PDF object is stored.
type xrefEntryKind int

const (
	xrefEntryFree     xrefEntryKind = 0 // free object slot
	xrefEntryDirect   xrefEntryKind = 1 // uncompressed object at file offset
	xrefEntryInObjStm xrefEntryKind = 2 // compressed object inside ObjStm
)

// xrefEntry describes a single object's location within a PDF file.
type xrefEntry struct {
	Kind       xrefEntryKind
	Offset     int64  // byte offset from file start (Kind==xrefEntryDirect)
	ObjStmNum  int64  // object number of containing ObjStm (Kind==xrefEntryInObjStm)
	IndexInStm int    // zero-based index within ObjStm (Kind==xrefEntryInObjStm)
	Generation int
}

// xrefIndex maps object numbers to their storage location.
// Later revisions override earlier ones (last-write-wins).
type xrefIndex map[int64]xrefEntry

// buildXRefIndex constructs a unified object location index by reading
// the xref chain starting at startXref and following /Prev links.
// Later revisions take precedence; entries are not overwritten.
func buildXRefIndex(f *os.File, startXref int64) (xrefIndex, error) {
	visited := map[int64]bool{}
	table := xrefIndex{}
	currentOffset := startXref

	for currentOffset != 0 && !visited[currentOffset] {
		visited[currentOffset] = true

		prevOffset, entries, objNums, err := parseXRefSectionAt(f, currentOffset)
		if err != nil {
			return nil, err
		}

		for i, entry := range entries {
			if i < len(objNums) {
				objNum := objNums[i]
				if _, exists := table[objNum]; !exists {
					table[objNum] = entry
				}
			}
		}

		currentOffset = prevOffset
	}

	return table, nil
}

// parseXRefSectionAt parses a single xref section (traditional table or stream)
// located at the given file offset. It returns the /Prev link offset, the parsed
// entries with their object numbers, or an error if the section cannot be decoded.
func parseXRefSectionAt(f *os.File, offset int64) (int64, []xrefEntry, []int64, error) {
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return 0, nil, nil, fmt.Errorf("seek to xref at %d: %w", offset, err)
	}

	buf := make([]byte, 20)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return 0, nil, nil, fmt.Errorf("read xref header at %d: %w", offset, err)
	}
	buf = buf[:n]

	header := strings.TrimSpace(string(buf))

	if strings.HasPrefix(header, "xref") {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return 0, nil, nil, err
		}
		return parseTraditionalXref(f)
	}

	fields := strings.Fields(header)
	if len(fields) >= 3 && fields[2] == "obj" {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return 0, nil, nil, err
		}
		return parseXRefStream(f, offset)
	}

	return 0, nil, nil, fmt.Errorf("unknown xref type at offset %d: %q", offset, header)
}

// discoverXRefOffsets walks the xref chain starting at startXref and returns
// the list of section offsets from newest to oldest. It parses just enough of
// each section to extract the /Prev link.
func discoverXRefOffsets(f *os.File, startXref int64) ([]int64, error) {
	var offsets []int64
	visited := map[int64]bool{}
	current := startXref

	for current != 0 && !visited[current] {
		visited[current] = true
		offsets = append(offsets, current)

		prevOffset, _, _, err := parseXRefSectionAt(f, current)
		if err != nil {
			return offsets, fmt.Errorf("discover xref offsets at %d: %w", current, err)
		}
		current = prevOffset
	}

	return offsets, nil
}

// parseTraditionalXref parses a traditional xref table and returns the Prev offset,
// slice of entries, and corresponding object numbers.
func parseTraditionalXref(f *os.File) (int64, []xrefEntry, []int64, error) {
	rd := bufio.NewReader(f)

	// Skip "xref" line
	line, err := rd.ReadString('\n')
	if err != nil {
		return 0, nil, nil, fmt.Errorf("read xref marker: %w", err)
	}
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "xref") {
		return 0, nil, nil, fmt.Errorf("expected xref marker, got %q", line)
	}

	var entries []xrefEntry
	var objNums []int64
	var prevOffset int64

	// Parse subsections until we hit "trailer"
	for {
		line, err = rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, nil, nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		lineTrimmed := strings.TrimSpace(line)

		if lineTrimmed == "trailer" {
			// Read trailer dict to find Prev
			trailerLine, err := rd.ReadString('\n')
			if err != nil && err != io.EOF {
				return 0, nil, nil, err
			}
			trailerLine = strings.TrimRight(trailerLine, "\r\n")
			openBrackets := strings.Count(trailerLine, "<<") - strings.Count(trailerLine, ">>")
			for openBrackets > 0 {
				nextLine, err := rd.ReadString('\n')
				if err == io.EOF {
					break
				}
				if err != nil {
					return 0, nil, nil, err
				}
				nextLine = strings.TrimRight(nextLine, "\r\n")
				trailerLine += nextLine
				openBrackets += strings.Count(nextLine, "<<") - strings.Count(nextLine, ">>")
			}

			// Extract /Prev from trailer dict
			idx := strings.Index(trailerLine, "/Prev ")
			if idx >= 0 {
				rest := trailerLine[idx+6:]
				endIdx := 0
				for endIdx < len(rest) && rest[endIdx] >= '0' && rest[endIdx] <= '9' {
					endIdx++
				}
				if endIdx > 0 {
					prevOffset, _ = strconv.ParseInt(rest[:endIdx], 10, 64)
				}
			}
			break
		}

		if lineTrimmed == "" {
			continue
		}

		// Parse subsection header: "start count"
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		start, _ := strconv.ParseInt(fields[0], 10, 64)
		count, _ := strconv.ParseInt(fields[1], 10, 64)

		// Read exactly count entries (each entry is 20 bytes: "OOOOOOOOOO GGGGG n \n")
		for i := int64(0); i < count; i++ {
			entryLine, err := rd.ReadString('\n')
			if err != nil && err != io.EOF {
				return 0, nil, nil, err
			}

			// Entry format: "OOOOOOOOOO GGGGG n \n" or "OOOOOOOOOO GGGGG f \n"
			// Exactly 20 characters
			entryLine = strings.TrimRight(entryLine, "\r\n")
			if len(entryLine) < 18 {
				continue
			}

			// Parse offset (10 digits)
			offsetStr := strings.TrimSpace(entryLine[0:10])
			offset, _ := strconv.ParseInt(offsetStr, 10, 64)

			// Parse generation (5 digits)
			genStr := strings.TrimSpace(entryLine[11:16])
			gen, _ := strconv.ParseInt(genStr, 10, 64)

			// Type is at position 17
			typ := 'f'
			if len(entryLine) > 17 {
				typ = rune(entryLine[17])
			}

			objNum := start + i
			var entry xrefEntry
			if typ == 'f' {
				entry = xrefEntry{Kind: xrefEntryFree, Generation: int(gen)}
			} else {
				entry = xrefEntry{Kind: xrefEntryDirect, Offset: offset, Generation: int(gen)}
			}
			entries = append(entries, entry)
			objNums = append(objNums, objNum)
		}
	}

	return prevOffset, entries, objNums, nil
}

// parseXRefStream parses an xref stream and returns the Prev offset,
// slice of entries, and corresponding object numbers.
func parseXRefStream(f *os.File, xrefOffset int64) (int64, []xrefEntry, []int64, error) {
	_, err := f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return 0, nil, nil, err
	}

	rd := bufio.NewReader(f)

	// Read object header "N G obj"
	headerLine, err := rd.ReadString('\n')
	if err != nil {
		return 0, nil, nil, fmt.Errorf("read xref stream header: %w", err)
	}
	rawConsumed := len(headerLine)
	headerLine = strings.TrimRight(headerLine, "\r\n")
	if !strings.Contains(headerLine, " obj") {
		return 0, nil, nil, fmt.Errorf("expected xref stream obj header, got %q", headerLine)
	}

	// Read stream dictionary until ">>stream"
	var dictLines []string
	streamFound := false
	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, nil, nil, err
		}
		rawConsumed += len(line)
		line = strings.TrimRight(line, "\r\n")
		dictLines = append(dictLines, line)

		// Check for stream marker
		combined := strings.Join(dictLines, "")
		if strings.Contains(combined, ">>stream") || strings.Contains(combined, ">>\nstream") {
			streamFound = true
			break
		}
		if strings.HasPrefix(line, "stream") {
			streamFound = true
			break
		}
	}

	dictStr := strings.Join(dictLines, "")

	// Extract /W (field widths)
	w0, w1, w2 := 1, 2, 1 // default field widths
	wStart := strings.Index(dictStr, "/W[")
	if wStart < 0 {
		wStart = strings.Index(dictStr, "/W [")
	}
	if wStart >= 0 {
		wEnd := strings.Index(dictStr[wStart:], "]")
		if wEnd > 0 {
			wStr := dictStr[wStart : wStart+wEnd+1]
			var widths []int
			// Parse integers from /W[...]
			numStr := ""
			for _, ch := range wStr {
				if ch >= '0' && ch <= '9' {
					numStr += string(ch)
				} else if numStr != "" {
					w, _ := strconv.Atoi(numStr)
					widths = append(widths, w)
					numStr = ""
				}
			}
			if numStr != "" {
				w, _ := strconv.Atoi(numStr)
				widths = append(widths, w)
			}
			if len(widths) >= 3 {
				w0, w1, w2 = widths[0], widths[1], widths[2]
			}
		}
	}
	entrySize := w0 + w1 + w2
	if entrySize == 0 {
		entrySize = 4 // default
	}

	// Extract /Size
	size := 0
	sizeStart := strings.Index(dictStr, "/Size ")
	if sizeStart >= 0 {
		sizeEnd := sizeStart + 6
		for sizeEnd < len(dictStr) && dictStr[sizeEnd] >= '0' && dictStr[sizeEnd] <= '9' {
			sizeEnd++
		}
		size, _ = strconv.Atoi(dictStr[sizeStart+6 : sizeEnd])
	}

	// Extract /Index (optional, pairs of start,count)
	var indices [][2]int // pairs of (start, count)
	indexStart := strings.Index(dictStr, "/Index[")
	if indexStart < 0 {
		indexStart = strings.Index(dictStr, "/Index [")
	}
	if indexStart >= 0 {
		indexEnd := strings.Index(dictStr[indexStart:], "]")
		if indexEnd > 0 {
			indexStr := dictStr[indexStart : indexStart+indexEnd+1]
			// Parse pairs of integers
			numStr := ""
			var nums []int
			for _, ch := range indexStr {
				if ch >= '0' && ch <= '9' {
					numStr += string(ch)
				} else if numStr != "" {
					n, _ := strconv.Atoi(numStr)
					nums = append(nums, n)
					numStr = ""
				}
			}
			if numStr != "" {
				n, _ := strconv.Atoi(numStr)
				nums = append(nums, n)
			}
			for i := 0; i+1 < len(nums); i += 2 {
				indices = append(indices, [2]int{nums[i], nums[i+1]})
			}
		}
	}
	if len(indices) == 0 {
		indices = append(indices, [2]int{0, size})
	}

	// Extract /Length
	streamLen := 0
	lengthStart := strings.Index(dictStr, "/Length ")
	if lengthStart >= 0 {
		lengthEnd := lengthStart + 8
		for lengthEnd < len(dictStr) && dictStr[lengthEnd] >= '0' && dictStr[lengthEnd] <= '9' {
			lengthEnd++
		}
		streamLen, _ = strconv.Atoi(dictStr[lengthStart+8 : lengthEnd])
	}

	// Extract /Prev
	var prevOffset int64
	prevStart := strings.Index(dictStr, "/Prev ")
	if prevStart >= 0 {
		prevEnd := prevStart + 6
		for prevEnd < len(dictStr) && dictStr[prevEnd] >= '0' && dictStr[prevEnd] <= '9' {
			prevEnd++
		}
		prevOffset, _ = strconv.ParseInt(dictStr[prevStart+6:prevEnd], 10, 64)
	}

	// Read stream data by seeking directly to the computed position
	if streamLen <= 0 || streamLen > 10000000 {
		return prevOffset, nil, nil, fmt.Errorf("invalid xref stream length: %d", streamLen)
	}

	if !streamFound {
		return prevOffset, nil, nil, fmt.Errorf("stream marker not found in xref stream dictionary")
	}

	// rawConsumed tracked the exact number of bytes read from xrefOffset through
	// the "stream" keyword and its trailing newline, so the file position for the
	// compressed payload is xrefOffset + rawConsumed.
	streamDataOffset := xrefOffset + int64(rawConsumed)
	if _, err := f.Seek(streamDataOffset, io.SeekStart); err != nil {
		return prevOffset, nil, nil, fmt.Errorf("seek to xref stream data at %d: %w", streamDataOffset, err)
	}

	streamData := make([]byte, streamLen)
	if _, err := io.ReadFull(f, streamData); err != nil {
		return prevOffset, nil, nil, fmt.Errorf("read xref stream data (%d bytes): %w", streamLen, err)
	}

	// Decompress
	r, err := zlib.NewReader(bytes.NewReader(streamData))
	if err != nil {
		return prevOffset, nil, nil, fmt.Errorf("decompress xref stream: %w", err)
	}
	decompressed, err := io.ReadAll(r)
	if err != nil {
		return prevOffset, nil, nil, fmt.Errorf("read decompressed xref stream: %w", err)
	}

	// Parse entries
	var entries []xrefEntry
	var objNums []int64
	dataIdx := 0
	for _, idx := range indices {
		start := idx[0]
		count := idx[1]
		for i := 0; i < count; i++ {
			if dataIdx+entrySize > len(decompressed) {
				break
			}
			entry := decompressed[dataIdx : dataIdx+entrySize]
			dataIdx += entrySize

			objNum := int64(start + i)
			typ := int(readUintFromBytes(entry[0:w0]))

			var xrefEnt xrefEntry
			switch typ {
			case 0:
				xrefEnt = xrefEntry{Kind: xrefEntryFree}
			case 1:
				offset := readUintFromBytes(entry[w0 : w0+w1])
				gen := int(readUintFromBytes(entry[w0+w1:]))
				xrefEnt = xrefEntry{Kind: xrefEntryDirect, Offset: int64(offset), Generation: gen}
			case 2:
				objStmNum := int64(readUintFromBytes(entry[w0 : w0+w1]))
				idxInStm := int(readUintFromBytes(entry[w0+w1:]))
				xrefEnt = xrefEntry{Kind: xrefEntryInObjStm, ObjStmNum: objStmNum, IndexInStm: idxInStm}
			default:
				xrefEnt = xrefEntry{Kind: xrefEntryFree}
			}
			entries = append(entries, xrefEnt)
			objNums = append(objNums, objNum)
		}
	}

	return prevOffset, entries, objNums, nil
}

// readUintFromBytes reads a big-endian unsigned integer from b (1-8 bytes).
func readUintFromBytes(b []byte) int64 {
	var result int64
	for _, v := range b {
		result = (result << 8) | int64(v)
	}
	return result
}

// resolveObject looks up objNum in the index and returns the raw object bytes.
// For Kind==xrefEntryDirect: reads directly at Offset.
// For Kind==xrefEntryInObjStm: decompresses the ObjStm and extracts the object.
func resolveObject(f *os.File, idx xrefIndex, objNum int64) ([]byte, error) {
	entry, ok := idx[objNum]
	if !ok || entry.Kind == xrefEntryFree {
		return nil, fmt.Errorf("object %d not found", objNum)
	}

	switch entry.Kind {
	case xrefEntryDirect:
		// Read object directly at Offset
		return readObjectAt(f, entry.Offset)

	case xrefEntryInObjStm:
		// Resolve from compressed object stream
		return resolveFromObjStm(f, idx, entry.ObjStmNum, entry.IndexInStm)

	default:
		return nil, fmt.Errorf("unsupported xref entry kind for object %d", objNum)
	}
}

// readObjectAt reads a PDF object starting at the given file offset.
// It reads until "endobj" is encountered and returns the raw object bytes.
func readObjectAt(f *os.File, offset int64) ([]byte, error) {
	_, err := f.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("seek to object at %d: %w", offset, err)
	}

	rd := bufio.NewReader(f)
	var obj bytes.Buffer

	for {
		line, err := rd.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		obj.WriteString(line)

		trimmed := strings.TrimSpace(line)
		if trimmed == "endobj" || strings.HasPrefix(trimmed, "endobj") {
			break
		}
	}

	return obj.Bytes(), nil
}

// ValidateDeep performs comprehensive PDF structure validation beyond basic header checks.
// It verifies xref table/stream integrity, object accessibility, trailer validity,
// and cross-reference consistency across the entire document.
func ValidateDeep(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("ValidateDeep: stat: %w", err)
	}
	if info.Size() < 20 {
		return fmt.Errorf("ValidateDeep: file too small for valid PDF")
	}

	xrefOffset, err := readStartxref(f)
	if err != nil {
		return fmt.Errorf("ValidateDeep: %w", err)
	}

	idx, err := buildXRefIndex(f, xrefOffset)
	if err != nil {
		return fmt.Errorf("ValidateDeep: build xref index: %w", err)
	}

	if len(idx) == 0 {
		return fmt.Errorf("ValidateDeep: no objects found in xref index")
	}

	for objNum := range idx {
		if objNum == 0 {
			continue
		}
		_, err := resolveObject(f, idx, objNum)
		if err != nil {
			return fmt.Errorf("ValidateDeep: object %d not resolvable: %w", objNum, err)
		}
	}

	trailerDict, isXRefStream, err := readTrailerDictFromFile(f, xrefOffset)
	if err != nil {
		return fmt.Errorf("ValidateDeep: trailer read: %w", err)
	}
	if trailerDict == "" {
		return fmt.Errorf("ValidateDeep: trailer dictionary not found")
	}

	trailer, err := ParseDictContent(trailerDict)
	if err != nil {
		return fmt.Errorf("ValidateDeep: parse trailer: %w", err)
	}

	if _, ok := DictGetRef(trailer, "Root"); !ok && !isXRefStream {
		return fmt.Errorf("ValidateDeep: /Root reference not found in trailer")
	}

	if isXRefStream {
		return nil
	}

	rootRefStr, err := readTrailerRootRef(f, xrefOffset)
	if err != nil {
		return fmt.Errorf("ValidateDeep: get root ref: %w", err)
	}
	catalogObj, err := readObject(f, rootRefStr)
	if err != nil {
		return fmt.Errorf("ValidateDeep: catalog object unreadable: %w", err)
	}
	catalogDict, err := extractDictFromObject(catalogObj)
	if err != nil {
		return fmt.Errorf("ValidateDeep: catalog dict: %w", err)
	}
	if typ, ok := DictGetName(catalogDict, "Type"); !ok || typ != "Catalog" {
		return fmt.Errorf("ValidateDeep: root is not /Type /Catalog")
	}

	return nil
}

func readTrailerDictFromFile(f *os.File, xrefOffset int64) (string, bool, error) {
	_, err := f.Seek(xrefOffset, io.SeekStart)
	if err != nil {
		return "", false, err
	}
	rd := bufio.NewReader(f)
	return readTrailerDictLines(rd)
}

// resolveFromObjStm resolves an object from a compressed object stream.
// It first resolves the ObjStm itself, decompresses it, then extracts the
// object at the specified index.
func resolveFromObjStm(f *os.File, idx xrefIndex, objStmNum int64, indexInStm int) ([]byte, error) {
	// Get the ObjStm entry
	stmEntry, ok := idx[objStmNum]
	if !ok {
		return nil, fmt.Errorf("object stream %d not found in xref", objStmNum)
	}
	if stmEntry.Kind != xrefEntryDirect {
		return nil, fmt.Errorf("object stream %d is not a direct object", objStmNum)
	}

	// Read the ObjStm object
	stmData, err := readObjectAt(f, stmEntry.Offset)
	if err != nil {
		return nil, fmt.Errorf("read object stream %d: %w", objStmNum, err)
	}

	// Extract stream content
	// ObjStm format: << /N count /First offset-of-index >> stream
	// followed by zlib-compressed data containing index + objects
	stmStr := string(stmData)

	// Find /N (number of objects)
	nStart := strings.Index(stmStr, "/N ")
	if nStart < 0 {
		nStart = strings.Index(stmStr, "/N=")
	}
	numObjs := 0
	if nStart >= 0 {
		nEnd := nStart + 3
		for nEnd < len(stmStr) && stmStr[nEnd] >= '0' && stmStr[nEnd] <= '9' {
			nEnd++
		}
		numObjs, _ = strconv.Atoi(stmStr[nStart+3 : nEnd])
	}

	// Find /First (offset to index within decompressed stream)
	firstOffset := 0
	firstStart := strings.Index(stmStr, "/First ")
	if firstStart < 0 {
		firstStart = strings.Index(stmStr, "/First=")
	}
	if firstStart >= 0 {
		firstEnd := firstStart + 7
		for firstEnd < len(stmStr) && stmStr[firstEnd] >= '0' && stmStr[firstEnd] <= '9' {
			firstEnd++
		}
		firstOffset, _ = strconv.Atoi(stmStr[firstStart+7 : firstEnd])
	}

	// Find stream data
	streamIdx := strings.Index(stmStr, ">>stream")
	if streamIdx < 0 {
		streamIdx = strings.Index(stmStr, ">>\nstream")
	}
	if streamIdx < 0 {
		return nil, fmt.Errorf("object stream %d: stream marker not found", objStmNum)
	}
	dataStart := streamIdx + 9
	if dataStart < len(stmStr) && stmStr[dataStart] == '\n' {
		dataStart++
	} else if dataStart < len(stmStr) && stmStr[dataStart] == '\r' {
		dataStart++
		if dataStart < len(stmStr) && stmStr[dataStart] == '\n' {
			dataStart++
		}
	}

	// Find endstream
	endStreamIdx := strings.Index(stmStr[dataStart:], "endstream")
	if endStreamIdx < 0 {
		return nil, fmt.Errorf("object stream %d: endstream not found", objStmNum)
	}
	streamData := []byte(stmStr[dataStart : dataStart+endStreamIdx])

	// Decompress
	r, err := zlib.NewReader(bytes.NewReader(streamData))
	if err != nil {
		return nil, fmt.Errorf("decompress object stream %d: %w", objStmNum, err)
	}
	decompressed, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read decompressed object stream %d: %w", objStmNum, err)
	}

	// Parse the index: first /N pairs of (objNum, offset)
	// Each pair is two integers
	indexData := string(decompressed[:firstOffset])
	var objOffsets []int
	var objNumbers []int
	fields := strings.Fields(indexData)
	for i := 0; i+1 < len(fields) && len(objNumbers) < numObjs; i += 2 {
		objNum, _ := strconv.Atoi(fields[i])
		offset, _ := strconv.Atoi(fields[i+1])
		objNumbers = append(objNumbers, objNum)
		objOffsets = append(objOffsets, offset)
	}

	// Extract the requested object
	if indexInStm < 0 || indexInStm >= len(objOffsets) {
		return nil, fmt.Errorf("object stream %d: index %d out of range (have %d objects)", objStmNum, indexInStm, len(objOffsets))
	}

	start := objOffsets[indexInStm]
	var end int
	if indexInStm+1 < len(objOffsets) {
		end = objOffsets[indexInStm+1]
	} else {
		end = len(decompressed) - firstOffset
	}

	// Adjust for the offset of data after the index
	actualStart := firstOffset + start
	actualEnd := firstOffset + end
	if actualStart > len(decompressed) {
		return nil, fmt.Errorf("object stream %d: start offset %d exceeds decompressed length %d", objStmNum, actualStart, len(decompressed))
	}
	if actualEnd > len(decompressed) {
		actualEnd = len(decompressed)
	}

	return decompressed[actualStart:actualEnd], nil
}
