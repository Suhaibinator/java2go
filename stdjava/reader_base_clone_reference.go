package stdjava

// CloneReaderBaseReference copies the represented Java Reader state without
// invoking a constructor, touching a monitor, or rebinding a self-valued lock.
// The native carrier must exist; a nullable protected Lock is copied verbatim.
func CloneReaderBaseReference(base *ReaderBase) *ReaderBase {
	ReferenceRequireNonNull(base)
	return &ReaderBase{Lock: base.Lock}
}
