package stdjava

// javaSingleByteOwnedLength counts one output byte per Java input character.
// A valid surrogate pair is one unmappable character; isolated surrogates are
// individual malformed characters. A conservative bound retains the caller's
// legacy fallback for sizes that cannot use the owned path.
func javaSingleByteOwnedLength(units []uint16, limit int) (int, bool) {
	if limit < 0 {
		return 0, false
	}
	total := 0
	for index := 0; index < len(units); index++ {
		if total == limit {
			return 0, false
		}
		total++
		if units[index] >= 0xd800 && units[index] <= 0xdbff && index+1 < len(units) && units[index+1] >= 0xdc00 && units[index+1] <= 0xdfff {
			index++
		}
	}
	return total, true
}

// encodeJavaSingleByteOwnedSigned owns its final mutable byte-array payload.
// ASCII maps units above 127 to '?'; Latin1 maps units above 255 to '?'. Both
// replace a valid supplementary pair once and each isolated surrogate once.
func encodeJavaSingleByteOwnedSigned(units []uint16, ascii bool) ([]int8, bool) {
	limit := int(^uint(0) >> 1)
	if limit > 2147483647 {
		limit = 2147483647
	}
	length, ok := javaSingleByteOwnedLength(units, limit)
	if !ok {
		return nil, false
	}
	maximum := uint16(255)
	if ascii {
		maximum = 127
	}
	encoded := make([]int8, length)
	offset := 0
	for index := 0; index < len(units); index++ {
		unit := units[index]
		if unit >= 0xd800 && unit <= 0xdbff && index+1 < len(units) && units[index+1] >= 0xdc00 && units[index+1] <= 0xdfff {
			index++
			unit = '?'
		}
		if unit > maximum {
			unit = '?'
		}
		encoded[offset] = int8(unit)
		offset++
	}
	return encoded, true
}
