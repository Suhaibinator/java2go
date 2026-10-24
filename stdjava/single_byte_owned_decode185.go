package stdjava

// decodeJavaSingleByteOwned decodes into exclusively new immutable UTF16
// storage. No input storage is retained, and even empty content gets a fresh
// JavaString wrapper. ASCII replaces each non-ASCII byte with one U+FFFD;
// Latin1 widens all eight input bits into one unit.
func decodeJavaSingleByteOwned(input []int8, ascii bool) *JavaString {
	units := make([]uint16, len(input))
	for index, value := range input {
		unit := uint16(byte(value))
		if ascii && unit >= 128 {
			unit = 0xfffd
		}
		units[index] = unit
	}
	return &JavaString{units: units}
}
