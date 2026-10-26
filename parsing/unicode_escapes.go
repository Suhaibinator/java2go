package parsing

import (
	"bytes"
	"strconv"
)

// UnicodeSourceMap retains physical source positions when redundant 'u' bytes
// in eligible Java Unicode escapes are removed for tree-sitter's grammar.
// Canonical and Original are semantically identical Java source. This is not
// general Unicode translation: escapes remain escapes for the existing parser
// and literal lowering. No newline bytes are inserted or removed.
type UnicodeSourceMap struct {
	Original  []byte
	Canonical []byte
	offsets   []uint32
}

func (mapping *UnicodeSourceMap) OriginalOffset(offset uint32) uint32 {
	if mapping == nil {
		return offset
	}
	if int(offset) >= len(mapping.offsets) {
		return uint32(len(mapping.Original))
	}
	return mapping.offsets[offset]
}

// canonicalJavaUnicodeEscapes accepts the JLS repeated-u spelling at exactly
// the same raw-input backslashes that are eligible for Unicode translation.
// Eligibility depends on the translated trailing backslashes, with an escape
// result making the immediately following raw backslash eligible as well.
func canonicalJavaUnicodeEscapes(source []byte) ([]byte, *UnicodeSourceMap) {
	result := make([]byte, 0, len(source))
	offsets := make([]uint32, 0, len(source)+1)
	appendSource := func(start, end int) {
		for index := start; index < end; index++ {
			result = append(result, source[index])
			offsets = append(offsets, uint32(index))
		}
	}
	trailingBackslashes := 0
	lastWasEscape := false
	changed := false
	for index := 0; index < len(source); {
		if source[index] == '\\' && (lastWasEscape || trailingBackslashes%2 == 0) {
			digits := index + 1
			for digits < len(source) && source[digits] == 'u' {
				digits++
			}
			if digits > index+1 && digits+4 <= len(source) {
				if value, err := strconv.ParseUint(string(source[digits:digits+4]), 16, 16); err == nil {
					appendSource(index, index+2)
					appendSource(digits, digits+4)
					changed = changed || digits > index+2
					index = digits + 4
					if value == '\\' {
						trailingBackslashes++
					} else {
						trailingBackslashes = 0
					}
					lastWasEscape = true
					continue
				}
			}
		}
		if source[index] == '\\' {
			trailingBackslashes++
		} else {
			trailingBackslashes = 0
		}
		lastWasEscape = false
		appendSource(index, index+1)
		index++
	}
	if !changed {
		return source, nil
	}
	offsets = append(offsets, uint32(len(source)))
	return result, &UnicodeSourceMap{Original: source, Canonical: result, offsets: offsets}
}

func (file *SourceFile) canonicalizeUnicodeEscapes() {
	if file.UnicodeSource != nil && bytes.Equal(file.Source, file.UnicodeSource.Canonical) {
		return
	}
	file.Source, file.UnicodeSource = canonicalJavaUnicodeEscapes(file.Source)
}
