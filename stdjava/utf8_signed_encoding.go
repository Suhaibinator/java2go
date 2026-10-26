package stdjava

// javaUTF8OwnedLength counts the existing Java UTF8 replacement policy without
// allocating. Refusing a conservative limit leaves the caller's legacy path in
// charge of unusually large inputs and its existing failure behavior.
func javaUTF8OwnedLength(units []uint16, limit int) (int, bool) {
	if limit < 0 {
		return 0, false
	}
	total := 0
	for index := 0; index < len(units); index++ {
		unit := units[index]
		width := 1
		switch {
		case unit < 0x80:
		case unit < 0x800:
			width = 2
		case unit >= 0xd800 && unit <= 0xdbff:
			if index+1 < len(units) && units[index+1] >= 0xdc00 && units[index+1] <= 0xdfff {
				width = 4
				index++
			}
		case unit >= 0xdc00 && unit <= 0xdfff:
		default:
			width = 3
		}
		if width > limit-total {
			return 0, false
		}
		total += width
	}
	return total, true
}

// encodeJavaUTF8OwnedSigned returns exclusively owned final byte-array storage.
// JavaString units are immutable; no public output borrows or shares this slice.
func encodeJavaUTF8OwnedSigned(units []uint16) ([]int8, bool) {
	limit := int(^uint(0) >> 1)
	if limit > 2147483647 {
		limit = 2147483647
	}
	length, ok := javaUTF8OwnedLength(units, limit)
	if !ok {
		return nil, false
	}
	encoded := make([]int8, length)
	offset := 0
	for index := 0; index < len(units); index++ {
		unit := units[index]
		switch {
		case unit < 0x80:
			encoded[offset] = int8(unit)
			offset++
		case unit < 0x800:
			encoded[offset] = int8(0xc0 | unit>>6)
			encoded[offset+1] = int8(0x80 | unit&0x3f)
			offset += 2
		case unit >= 0xd800 && unit <= 0xdbff:
			if index+1 < len(units) && units[index+1] >= 0xdc00 && units[index+1] <= 0xdfff {
				point := uint32(0x10000) + (uint32(unit-0xd800) << 10) + uint32(units[index+1]-0xdc00)
				encoded[offset] = int8(0xf0 | point>>18)
				encoded[offset+1] = int8(0x80 | (point>>12)&0x3f)
				encoded[offset+2] = int8(0x80 | (point>>6)&0x3f)
				encoded[offset+3] = int8(0x80 | point&0x3f)
				offset += 4
				index++
			} else {
				encoded[offset] = '?'
				offset++
			}
		case unit >= 0xdc00 && unit <= 0xdfff:
			encoded[offset] = '?'
			offset++
		default:
			encoded[offset] = int8(0xe0 | unit>>12)
			encoded[offset+1] = int8(0x80 | (unit>>6)&0x3f)
			encoded[offset+2] = int8(0x80 | unit&0x3f)
			offset += 3
		}
	}
	return encoded, true
}
