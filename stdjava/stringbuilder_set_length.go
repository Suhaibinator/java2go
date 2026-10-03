package stdjava

import "fmt"

// SetLength edits the UTF16 unit count, preserving retained units and filling
// growth with NUL. Previously truncated storage must not reappear on regrowth.
func (b *StringBuilder) SetLength(newLength int32) {
	ReferenceRequireNonNull(b)
	if newLength < 0 {
		panic(NewStringIndexOutOfBoundsException(fmt.Sprintf("String index out of range: %d", newLength)))
	}
	previous, length := len(b.buf), int(newLength)
	if length <= cap(b.buf) {
		b.buf = b.buf[:length]
		if length > previous {
			clear(b.buf[previous:])
		}
		return
	}
	b.buf = append(b.buf, make([]rune, length-previous)...)
}
