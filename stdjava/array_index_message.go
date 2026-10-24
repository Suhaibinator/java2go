package stdjava

import "strconv"

// Bytecode array accesses report the saved signed index and immediate length.
// Reflection, arraycopy, and explicit exception constructors retain their own
// service-specific messages at their existing call sites.
func arrayIndexOutOfBoundsMessage(index int64, length int) string {
	return "Index " + strconv.FormatInt(index, 10) + " out of bounds for length " + strconv.Itoa(length)
}
