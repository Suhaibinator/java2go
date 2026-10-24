package stdjava

import "strconv"

// IntegerToHexString renders the unsigned 32-bit representation of a Java int.
func IntegerToHexString(value int32) string {
	return strconv.FormatUint(uint64(uint32(value)), 16)
}
