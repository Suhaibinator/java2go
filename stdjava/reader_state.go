package stdjava

// ReaderState returns the existing native carrier without altering its mutable
// lock. Source constructors eagerly install state with the complete Java self.
func (base *ReaderBase) JavaReaderState() *ReaderBase { return base }
func ReaderState(reader Reader) *ReaderBase {
	ReferenceRequireNonNull(reader)
	if view, ok := reader.(interface{ JavaReaderState() *ReaderBase }); ok {
		state := view.JavaReaderState()
		ReferenceRequireNonNull(state)
		return state
	}
	panic(NewClassCastException("Reader implementation lacks base state"))
}
