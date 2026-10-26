package stdjava

// ToJavaString snapshots the builder into a fresh reference-bearing String.
// JDK21 StringBuilder.toString constructs a new String even for equal repeated
// snapshots. Each rune in buf is one UTF16 unit, not a Unicode scalar value.
// The existing native String() conversion is deliberately unchanged.
func (b *StringBuilder) ToJavaString() *JavaString {
	ReferenceRequireNonNull(b)
	units := make([]uint16, len(b.buf))
	for index, unit := range b.buf {
		units[index] = uint16(unit)
	}
	return &JavaString{units: units}
}
