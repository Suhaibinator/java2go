package stdjava

// JavaStringCharsStream returns the canonical String's UTF16 units as Java ints.
// It checks the receiver when chars() is invoked and never decodes surrogates
// through a host string, preserving pairs, isolated units, and embedded NULs.
func JavaStringCharsStream(text *JavaString) Stream[int32] {
	ReferenceRequireNonNull(text)
	units := make([]int32, len(text.units))
	for index, unit := range text.units {
		units[index] = int32(unit)
	}
	return Stream[int32]{elements: units}
}
