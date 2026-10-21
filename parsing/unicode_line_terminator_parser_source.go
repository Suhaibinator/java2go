package parsing

import "strconv"

// Tree-sitter receives a private equal-width view of eligible Java Unicode
// line terminators. Original/canonical source bytes and their maps stay intact.
// Eligibility follows the pinned javac UnicodeReader previous-backslash and
// previous-valid-escape state; translated backslashes are never rescanned.
func unicodeLineTerminatorParserSource(source []byte) []byte {
	var view []byte
	wasBackslash, wasEscape := false, false
	for index := 0; index < len(source); {
		if source[index] == '\\' && (!wasBackslash || wasEscape) {
			digits := index + 1
			for digits < len(source) && source[digits] == 'u' {
				digits++
			}
			if digits > index+1 && digits+4 <= len(source) {
				if value, err := strconv.ParseUint(string(source[digits:digits+4]), 16, 16); err == nil {
					end := digits + 4
					if value == '\n' || value == '\r' {
						if view == nil {
							view = append([]byte(nil), source...)
						}
						for offset := index; offset < end-1; offset++ {
							view[offset] = ' '
						}
						// The grammar terminates line comments on LF. Java CR and LF
						// are lexically equivalent here; source bytes remain unchanged.
						view[end-1] = '\n'
					}
					wasEscape = true
					wasBackslash = value == '\\' && !wasBackslash
					index = end
					continue
				}
			}
			wasEscape = false
			wasBackslash = !wasBackslash
		} else {
			wasBackslash = false
			wasEscape = false
		}
		index++
	}
	if view == nil {
		return source
	}
	return view
}
