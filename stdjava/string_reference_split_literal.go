package stdjava

// String.split in JDK21 scans a literal UTF16 delimiter when the pattern is one
// non-meta character or an escaped non-ASCII-alphanumeric character. Surrogate
// delimiters retain regex matching so code-point composition stays unchanged.
func stringSplitLiteralDelimiter(regex *JavaString) (uint16, bool) {
	var delimiter uint16
	switch len(regex.units) {
	case 1:
		delimiter = regex.units[0]
		switch delimiter {
		case '.', '$', '|', '(', ')', '[', '{', '^', '?', '*', '+', '\\':
			return 0, false
		}
	case 2:
		if regex.units[0] != '\\' {
			return 0, false
		}
		delimiter = regex.units[1]
		if delimiter >= '0' && delimiter <= '9' || delimiter >= 'a' && delimiter <= 'z' || delimiter >= 'A' && delimiter <= 'Z' {
			return 0, false
		}
	default:
		return 0, false
	}
	return delimiter, delimiter < 0xd800 || delimiter > 0xdfff
}

// Count the result before allocating its array. Matching itself keeps no regex
// nodes, captures or temporary part slices; every returned String still uses
// the existing substring allocation and identity rules.
func stringSplitLiteralArray(text *JavaString, delimiter uint16, limit int32) *ReferenceArray {
	matches := 0
	for _, unit := range text.units {
		if unit == delimiter {
			if limit > 0 && matches == int(limit)-1 {
				break
			}
			matches++
		}
	}
	if matches == 0 {
		return ReferenceArrayLiteral(StringTypeID, text)
	}
	size := matches + 1
	end := len(text.units)
	if limit == 0 {
		for end > 0 && text.units[end-1] == delimiter {
			end--
		}
		if end == 0 {
			return NewReferenceArray(0, StringTypeID)
		}
		size -= len(text.units) - end
	}
	result := NewReferenceArray(size, StringTypeID)
	from, next := 0, 0
	for index := 0; index < size-1; index++ {
		for text.units[next] != delimiter {
			next++
		}
		result.elements[index] = text.Substring(int32(from), int32(next))
		next++
		from = next
	}
	result.elements[size-1] = text.Substring(int32(from), int32(end))
	return result
}
